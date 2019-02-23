package differ

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"sort"
	"strings"
	"sync"
	"time"

	"bitbucket.org/atlassian/vpcflow-diffd/pkg/domain"
	radix "github.com/armon/go-radix"
	"github.com/asecurityteam/vpcflow-diffd/pkg/domain"
)

// The set of keyed attributes of an edge. This are values which we won't expect to change between graph generations
var keyAttrs = map[string]bool{
	"govpc_accountID": true,
	"govpc_eniID":     true,
	"govpc_srcPort":   true,
	"govpc_dstPort":   true,
	"govpc_protocol":  true,
	"color":           true, // red/green represents status reject/accept
}

// DOTDiffer is a differ implementation which takes two DOT graphs, and generates a diff between the two
type DOTDiffer struct {
	Grapher domain.Grapher
}

// Diff generates the diff of two DOT graphs
func (d *DOTDiffer) Diff(ctx context.Context, diff domain.Diff) (io.ReadCloser, error) {

	// Note for readers: This function previously used the same DOT parsing library
	// as go-vpcflow uses to create the graphs. However, we started hitting OOM errors
	// when running on only a portion of our full VPC flow data for a 24 hour period.
	// The combination of reading two large graph files into memory and parsing them
	// began to exceed 16GiB for a few million records. As a stop-gap we have refactored
	// this method to use a radix/prefix tree rather than maps to track data and
	// repeatedly streaming the full data set through the system rather than load
	// and buffer. This is a trade-off of memory for bandwidth.
	//
	// Additionally, we've dropped the full lexer/parser in favor of working directly
	// with the string content of the file. As a result, we are susceptible to problems
	// cause by faulty or corrupted graph input.
	//
	// Both of these are temporary measures to ensure we can generate DIFF graphs while
	// we work on a more complete solution that scales beyond a fraction of our data
	// for 24 hours.

	prevChan := make(chan io.ReadCloser, 1)
	prevSourceChan := make(chan io.ReadCloser, 1)
	nextChan := make(chan io.ReadCloser, 1)
	nextSourceChan := make(chan io.ReadCloser, 1)
	errs := make(chan error, 4)
	wg := &sync.WaitGroup{}
	wg.Add(4)

	go d.getGraph(ctx, prevChan, errs, diff.PreviousStart, diff.PreviousStop, wg)
	go d.getGraph(ctx, prevSourceChan, errs, diff.PreviousStart, diff.PreviousStop, wg)
	go d.getGraph(ctx, nextChan, errs, diff.NextStart, diff.NextStop, wg)
	go d.getGraph(ctx, nextSourceChan, errs, diff.NextStart, diff.NextStop, wg)
	wg.Wait()

	close(errs)
	prevGraph := <-prevChan
	prevSourceGraph := <-prevSourceChan
	nextGraph := <-nextChan
	nextSourceGraph := <-nextSourceChan
	if nextGraph != nil {
		defer nextGraph.Close()
	}
	if nextSourceGraph != nil {
		defer nextSourceGraph.Close()
	}
	if prevGraph != nil {
		defer prevGraph.Close()
	}
	if prevSourceGraph != nil {
		defer prevSourceGraph.Close()
	}
	for err := range errs {
		if err != nil {
			return nil, err
		}
	}

	nodes := radix.New()
	prevReader := bufio.NewReader(prevGraph)
	prevSearch := radix.New()
	var err error
	var line string
	for line, err = prevReader.ReadString('\n'); err == nil; line, err = prevReader.ReadString('\n') {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, keyType, _ := lineKey(line)
		if keyType == lineTypeNode {
			nodes.Insert(key, line)
			continue
		}
		_, _ = prevSearch.Insert(key, nil)
	}
	if err != nil && err != io.EOF {
		return nil, err
	}

	nextReader := bufio.NewReader(nextGraph)
	nextSearch := radix.New()
	for line, err = nextReader.ReadString('\n'); err == nil; line, err = nextReader.ReadString('\n') {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, keyType, _ := lineKey(line)
		if keyType == lineTypeNode {
			nodes.Insert(key, line)
			continue
		}
		_, _ = nextSearch.Insert(key, nil)
	}
	if err != nil && err != io.EOF {
		return nil, err
	}

	output := bytes.NewBufferString("digraph {\n")
	nodesToShow := radix.New()
	nextSourceReader := bufio.NewReader(nextSourceGraph)
	for line, err = nextSourceReader.ReadString('\n'); err == nil; line, err = nextSourceReader.ReadString('\n') {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, keyType, edgeNodes := lineKey(line)
		_, prevFound := prevSearch.Get(key)
		if keyType == lineTypeNode {
			continue
		}
		if keyType == lineTypeEdge && !prevFound {
			for offset := range edgeNodes {
				nodesToShow.Insert(edgeNodes[offset], nil)
			}
			_, _ = output.WriteString(line[:len(line)-2]) // remove ] and newline".
			_, _ = output.WriteString("\\ndiff=ADDED\" govpc_diff=\"ADDED\"]\n")
		}
	}
	if err != nil && err != io.EOF {
		return nil, err
	}

	prevSourceReader := bufio.NewReader(prevSourceGraph)
	for line, err = prevSourceReader.ReadString('\n'); err == nil; line, err = prevSourceReader.ReadString('\n') {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, keyType, edgeNodes := lineKey(line)
		_, nextFound := nextSearch.Get(key)
		if keyType == lineTypeNode {
			continue
		}
		if keyType == lineTypeEdge && !nextFound {
			for offset := range edgeNodes {
				nodesToShow.Insert(edgeNodes[offset], nil)
			}
			_, _ = output.WriteString(line[:len(line)-2]) // remove ] and newline".
			_, _ = output.WriteString("\\ndiff=REMOVED\" govpc_diff=\"REMOVED\"]\n")
		}
	}
	if err != nil && err != io.EOF {
		return nil, err
	}

	nodesToShow.Walk(func(node string, _ interface{}) bool {
		nodeValue, _ := nodes.Get(node)
		_, _ = output.WriteString(nodeValue.(string))
		_, _ = output.WriteString("\n")
		return false
	})
	_, _ = output.WriteString("}")

	return ioutil.NopCloser(output), nil
}

type lineType uint

const (
	lineTypeNode lineType = 1
	lineTypeEdge lineType = 2
)

func lineKey(line string) (string, lineType, []string) {
	if strings.Contains(line, "->") {
		key, nodes := lineEdgeKey(line)
		return key, lineTypeEdge, nodes
	}
	return lineNodeKey(line), lineTypeNode, nil
}

// lineEdgeKey converts an edge written by the go-vpcflow graph component
// in to a unique key. The source format looks like:
//
// n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="20" govpc_bytes="1000" govpc_start="1418530010" govpc_end="1818530070" color=red label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=20\nbytes=1000\nstart=1418530010\nend=1818530070"]
//
// The output format looks like:
//
// n1723116139n172311621govpc_accountID="123456789010"govpc_eniID="eni-abc123de"govpc_srcPort="0"govpc_dstPort="80"govpc_protocol="6"color=red
//
// where the selected attributes are first sorted consistently. The returned
// slice contains the ID of nodes in the edge.
func lineEdgeKey(line string) (string, []string) {
	parts := strings.SplitN(line, " ", 4)
	attrs := strings.Split(parts[3][1:len(parts[3])-1], " ")
	key := bytes.NewBufferString(parts[0])
	_, _ = key.WriteString(parts[2])
	selectedAttrs := make([]string, 0, len(keyAttrs))
	for offset := range attrs {
		if keyAttrs[strings.Split(attrs[offset], "=")[0]] {
			selectedAttrs = append(selectedAttrs, attrs[offset])
		}
	}
	sort.Strings(selectedAttrs)
	for offset := range selectedAttrs {
		_, _ = key.WriteString(selectedAttrs[offset])
	}
	return key.String(), []string{parts[0], parts[2]}
}

// lineNodeKey returns the left-hand side of n172311622 [label="172.31.16.22"]
// which is how the go-vpcflow graph writes a node.
func lineNodeKey(line string) string {
	return strings.TrimSpace(strings.SplitN(strings.TrimSpace(line), " ", 2)[0])
}

func (d *DOTDiffer) getGraph(ctx context.Context, out chan io.ReadCloser, err chan error, start, stop time.Time, wg *sync.WaitGroup) {
	defer wg.Done()
	g, e := d.Grapher.Graph(ctx, start, stop)
	out <- g
	err <- e
}
