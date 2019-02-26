package differ

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	"github.com/asecurityteam/vpcflow-diffd/pkg/domain"
	"gonum.org/v1/gonum/graph/formats/dot"
	"gonum.org/v1/gonum/graph/formats/dot/ast"
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
	prevChan := make(chan io.ReadCloser, 1)
	nextChan := make(chan io.ReadCloser, 1)
	errs := make(chan error, 2)
	wg := &sync.WaitGroup{}
	wg.Add(2)

	go d.getGraph(ctx, prevChan, errs, diff.PreviousStart, diff.PreviousStop, wg)
	go d.getGraph(ctx, nextChan, errs, diff.NextStart, diff.NextStop, wg)
	wg.Wait()

	close(errs)
	prevGraph := <-prevChan
	nextGraph := <-nextChan
	if nextGraph != nil {
		defer nextGraph.Close()
	}
	if prevGraph != nil {
		defer prevGraph.Close()
	}
	for err := range errs {
		if err != nil {
			return nil, err
		}
	}

	prevEdges, prevNodes, err := parseGraph(prevGraph)
	if err != nil {
		return nil, err
	}
	nextEdges, nextNodes, err := parseGraph(nextGraph)
	if err != nil {
		return nil, err
	}
	// keep a map of nodes for deduping
	nodes := make(map[string]bool)
	removedEdges := graphDiff(prevEdges, nextEdges, "REMOVED", nodes)
	addedEdges := graphDiff(nextEdges, prevEdges, "ADDED", nodes)
	g := &ast.Graph{Directed: true}
	g.Stmts = append(g.Stmts, removedEdges...)
	g.Stmts = append(g.Stmts, addedEdges...)
	for nodeID := range nodes {
		var stmt *ast.NodeStmt
		var ok bool
		stmt, ok = prevNodes[nodeID]
		if !ok {
			stmt = nextNodes[nodeID]
		}
		g.Stmts = append(g.Stmts, stmt)
	}
	return ioutil.NopCloser(bytes.NewReader([]byte(g.String()))), nil
}

// graphDiff takes the difference of edges in a and b. In other words it finds the set of edges that
// exist in a, but not in b. As it finds these edges it adds the provided tag to the edge's label which
// better describes the meaning behind the difference. As well as finding the difference in edges, a map
// of nodes is also used to track unique node IDs which appear in the diff.
func graphDiff(a, b map[string]*ast.EdgeStmt, tag string, nodes map[string]bool) []ast.Stmt {
	diff := make([]ast.Stmt, 0)
	for k, v := range a {
		_, ok := b[k]
		if ok { // edge is in both graphs
			continue
		}
		// modify the label to display "diff=<tag>" on the edge annotation
		label := v.Attrs[len(v.Attrs)-1].Val
		label = strings.Trim(label, `"`)
		label = fmt.Sprintf(`%s\ndiff=%s`, label, tag)
		v.Attrs[len(v.Attrs)-1].Val = fmt.Sprintf(`"%s"`, label)
		// also add tag as namespaced attribute for easy parsing by downstream consumers
		v.Attrs = append(v.Attrs, &ast.Attr{
			Key: "govpc_diff",
			Val: fmt.Sprintf(`"%s"`, tag),
		})
		diff = append(diff, v)
		nodes[v.From.String()] = true
		nodes[v.To.Vertex.String()] = true
	}
	return diff
}

// parse the graph into maps of edges and nodes for easy lookup
func parseGraph(graph io.Reader) (map[string]*ast.EdgeStmt, map[string]*ast.NodeStmt, error) {
	f, err := dot.Parse(graph)
	if err != nil {
		return nil, nil, err
	}
	edges := make(map[string]*ast.EdgeStmt)
	nodes := make(map[string]*ast.NodeStmt)
	for _, g := range f.Graphs {
		for _, stmt := range g.Stmts {
			switch v := stmt.(type) {
			case *ast.EdgeStmt:
				edges[edgeKey(v)] = v
			case *ast.NodeStmt:
				nodes[v.Node.ID] = v
			default:
				continue
			}
		}
	}
	return edges, nodes, nil
}

// return a key which uniquely identifies this edge
func edgeKey(edge *ast.EdgeStmt) string {
	key := fmt.Sprintf("%s_%s", edge.From.String(), edge.To.Vertex.String())
	for _, attr := range edge.Attrs {
		_, ok := keyAttrs[attr.Key]
		if !ok {
			continue
		}
		key = key + "_" + attr.Val
	}
	return key
}

func (d *DOTDiffer) getGraph(ctx context.Context, out chan io.ReadCloser, err chan error, start, stop time.Time, wg *sync.WaitGroup) {
	defer wg.Done()
	g, e := d.Grapher.Graph(ctx, start, stop)
	out <- g
	err <- e
}
