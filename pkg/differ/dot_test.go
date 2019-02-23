package differ

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"bitbucket.org/atlassian/vpcflow-diffd/pkg/domain"
	"github.com/asecurityteam/vpcflow-diffd/pkg/domain"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestPrevGraphError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := domain.Diff{
		PreviousStart: time.Now().Add(-1 * time.Hour),
		PreviousStop:  time.Now().Add(-1 * time.Hour),
		NextStart:     time.Now(),
		NextStop:      time.Now(),
	}

	grapherMock := NewMockGrapher(ctrl)
	grapherMock.EXPECT().Graph(gomock.Any(), d.PreviousStart, d.PreviousStop).Return(nil, errors.New("")).AnyTimes()
	grapherMock.EXPECT().Graph(gomock.Any(), d.NextStart, d.NextStop).Return(ioutil.NopCloser(bytes.NewReader([]byte(""))), nil).AnyTimes()

	differ := DOTDiffer{grapherMock}
	_, err := differ.Diff(context.Background(), d)
	assert.NotNil(t, err)
}

func TestNextGraphError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := domain.Diff{
		PreviousStart: time.Now().Add(-1 * time.Hour),
		PreviousStop:  time.Now().Add(-1 * time.Hour),
		NextStart:     time.Now(),
		NextStop:      time.Now(),
	}

	grapherMock := NewMockGrapher(ctrl)
	grapherMock.EXPECT().Graph(gomock.Any(), d.PreviousStart, d.PreviousStop).Return(ioutil.NopCloser(bytes.NewReader([]byte(""))), nil)
	grapherMock.EXPECT().Graph(gomock.Any(), d.PreviousStart, d.PreviousStop).Return(ioutil.NopCloser(bytes.NewReader([]byte(""))), nil)
	grapherMock.EXPECT().Graph(gomock.Any(), d.NextStart, d.NextStop).Return(nil, errors.New(""))
	grapherMock.EXPECT().Graph(gomock.Any(), d.NextStart, d.NextStop).Return(nil, errors.New(""))

	differ := DOTDiffer{grapherMock}
	_, err := differ.Diff(context.Background(), d)
	assert.NotNil(t, err)
}

func TestDiff(t *testing.T) {

	tc := []struct {
		Name     string
		Previous string
		Next     string
		Expected map[string]bool
	}{
		{
			Name: "added_node",
			Previous: `digraph {
				n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="20" govpc_bytes="1000" govpc_start="1418530010" govpc_end="1818530070" color=red label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=20\nbytes=1000\nstart=1418530010\nend=1818530070"]
				n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
				n172311621 -> n1723116139 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="80" govpc_dstPort="0" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=80\ndstPort=0\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
				n1723116139 [label="172.31.16.139"]
				n172311621 [label="172.31.16.21"]
			}`,
			Next: `digraph {
				n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="20" govpc_bytes="1000" govpc_start="1418530010" govpc_end="1818530070" color=red label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=20\nbytes=1000\nstart=1418530010\nend=1818530070"]
				n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
				n172311621 -> n1723116139 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="80" govpc_dstPort="0" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=80\ndstPort=0\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
				n172311621 -> n172311622 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=80\ndstPort=0\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
				n172311622 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="80" govpc_dstPort="0" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=80\ndstPort=0\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
				n1723116139 [label="172.31.16.139"]
				n172311621 [label="172.31.16.21"]
				n172311622 [label="172.31.16.22"]
			}`,
			Expected: map[string]bool{
				`digraph {`: true,
				`n172311621 -> n172311622 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=80\ndstPort=0\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070\ndiff=ADDED" govpc_diff="ADDED"]`: true,
				`n172311622 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="80" govpc_dstPort="0" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=80\ndstPort=0\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070\ndiff=ADDED" govpc_diff="ADDED"]`: true,
				`n172311621 [label="172.31.16.21"]`: true,
				`n172311622 [label="172.31.16.22"]`: true,
				`}`:                                 true,
			},
		},
		{
			Name: "added_port",
			Previous: `digraph {
				n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="20" govpc_bytes="1000" govpc_start="1418530010" govpc_end="1818530070" color=red label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=20\nbytes=1000\nstart=1418530010\nend=1818530070"]
				n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
				n172311621 -> n1723116139 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="80" govpc_dstPort="0" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=80\ndstPort=0\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
				n1723116139 [label="172.31.16.139"]
				n172311621 [label="172.31.16.21"]
			}`,
			Next: `digraph {
				n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="20" govpc_bytes="1000" govpc_start="1418530010" govpc_end="1818530070" color=red label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=20\nbytes=1000\nstart=1418530010\nend=1818530070"]
				n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
				n172311621 -> n1723116139 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="80" govpc_dstPort="0" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=80\ndstPort=0\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
				n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="22" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
				n172311621 -> n1723116139 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="22" govpc_dstPort="0" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=80\ndstPort=0\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
				n1723116139 [label="172.31.16.139"]
				n172311621 [label="172.31.16.21"]
			}`,
			Expected: map[string]bool{
				`digraph {`: true,
				`n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="22" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070\ndiff=ADDED" govpc_diff="ADDED"]`: true,
				`n172311621 -> n1723116139 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="22" govpc_dstPort="0" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=80\ndstPort=0\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070\ndiff=ADDED" govpc_diff="ADDED"]`: true,
				`n1723116139 [label="172.31.16.139"]`: true,
				`n172311621 [label="172.31.16.21"]`:   true,
				`}`:                                   true,
			},
		},
		{
			Name: "removed_node",
			Next: `digraph {
				n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="20" govpc_bytes="1000" govpc_start="1418530010" govpc_end="1818530070" color=red label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=20\nbytes=1000\nstart=1418530010\nend=1818530070"]
				n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
				n172311621 -> n1723116139 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="80" govpc_dstPort="0" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=80\ndstPort=0\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
				n1723116139 [label="172.31.16.139"]
				n172311621 [label="172.31.16.21"]
			}`,
			Previous: `digraph {
				n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="20" govpc_bytes="1000" govpc_start="1418530010" govpc_end="1818530070" color=red label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=20\nbytes=1000\nstart=1418530010\nend=1818530070"]
				n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
				n172311621 -> n1723116139 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="80" govpc_dstPort="0" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=80\ndstPort=0\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
				n172311621 -> n172311622 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=80\ndstPort=0\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
				n172311622 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="80" govpc_dstPort="0" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=80\ndstPort=0\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
				n1723116139 [label="172.31.16.139"]
				n172311621 [label="172.31.16.21"]
				n172311622 [label="172.31.16.22"]
			}`,
			Expected: map[string]bool{
				`digraph {`: true,
				`n172311621 -> n172311622 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=80\ndstPort=0\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070\ndiff=REMOVED" govpc_diff="REMOVED"]`: true,
				`n172311622 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="80" govpc_dstPort="0" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=80\ndstPort=0\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070\ndiff=REMOVED" govpc_diff="REMOVED"]`: true,
				`n172311621 [label="172.31.16.21"]`: true,
				`n172311622 [label="172.31.16.22"]`: true,
				`}`:                                 true,
			},
		},
		{
			Name: "removed_port",
			Next: `digraph {
				n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="20" govpc_bytes="1000" govpc_start="1418530010" govpc_end="1818530070" color=red label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=20\nbytes=1000\nstart=1418530010\nend=1818530070"]
				n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
				n172311621 -> n1723116139 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="80" govpc_dstPort="0" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=80\ndstPort=0\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
				n1723116139 [label="172.31.16.139"]
				n172311621 [label="172.31.16.21"]
			}`,
			Previous: `digraph {
				n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="20" govpc_bytes="1000" govpc_start="1418530010" govpc_end="1818530070" color=red label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=20\nbytes=1000\nstart=1418530010\nend=1818530070"]
				n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
				n172311621 -> n1723116139 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="80" govpc_dstPort="0" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=80\ndstPort=0\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
				n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="22" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
				n172311621 -> n1723116139 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="22" govpc_dstPort="0" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=80\ndstPort=0\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
				n1723116139 [label="172.31.16.139"]
				n172311621 [label="172.31.16.21"]
			}`,
			Expected: map[string]bool{
				`digraph {`: true,
				`n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="22" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070\ndiff=REMOVED" govpc_diff="REMOVED"]`: true,
				`n172311621 -> n1723116139 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="22" govpc_dstPort="0" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=80\ndstPort=0\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070\ndiff=REMOVED" govpc_diff="REMOVED"]`: true,
				`n1723116139 [label="172.31.16.139"]`: true,
				`n172311621 [label="172.31.16.21"]`:   true,
				`}`:                                   true,
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.Name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			d := domain.Diff{
				PreviousStart: time.Now().Add(-1 * time.Hour),
				PreviousStop:  time.Now().Add(-1 * time.Hour),
				NextStart:     time.Now(),
				NextStop:      time.Now(),
			}

			grapherMock := NewMockGrapher(ctrl)
			grapherMock.EXPECT().Graph(gomock.Any(), d.PreviousStart, d.PreviousStop).Return(ioutil.NopCloser(bytes.NewReader([]byte(tt.Previous))), nil)
			grapherMock.EXPECT().Graph(gomock.Any(), d.PreviousStart, d.PreviousStop).Return(ioutil.NopCloser(bytes.NewReader([]byte(tt.Previous))), nil)
			grapherMock.EXPECT().Graph(gomock.Any(), d.NextStart, d.NextStop).Return(ioutil.NopCloser(bytes.NewReader([]byte(tt.Next))), nil)
			grapherMock.EXPECT().Graph(gomock.Any(), d.NextStart, d.NextStop).Return(ioutil.NopCloser(bytes.NewReader([]byte(tt.Next))), nil)

			differ := DOTDiffer{grapherMock}
			out, err := differ.Diff(context.Background(), d)
			assert.Nil(t, err)
			reader := bufio.NewReader(out)
			var numLines int
			for {
				line, err := reader.ReadString('\n')
				if err == io.EOF && len(line) < 1 {
					break
				}
				numLines++
				line = strings.TrimSpace(line)
				_, found := tt.Expected[line]
				assert.True(t, found, fmt.Sprintf("Did not expect line: %s", line))
				tt.Expected[line] = false // we should only encouter each line in the digest once
			}
			assert.Equal(t, len(tt.Expected), numLines)
		})
	}
}

func BenchmarkDiff(b *testing.B) {

	tc := []struct {
		Name     string
		Previous string
		Next     string
	}{
		{
			Name: "added_node",
			Previous: `digraph {
					n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="20" govpc_bytes="1000" govpc_start="1418530010" govpc_end="1818530070" color=red label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=20\nbytes=1000\nstart=1418530010\nend=1818530070"]
					n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
					n172311621 -> n1723116139 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="80" govpc_dstPort="0" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=80\ndstPort=0\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
					n1723116139 [label="172.31.16.139"]
					n172311621 [label="172.31.16.21"]
				}`,
			Next: `digraph {
					n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="20" govpc_bytes="1000" govpc_start="1418530010" govpc_end="1818530070" color=red label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=20\nbytes=1000\nstart=1418530010\nend=1818530070"]
					n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
					n172311621 -> n1723116139 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="80" govpc_dstPort="0" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=80\ndstPort=0\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
					n172311621 -> n172311622 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=80\ndstPort=0\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
					n172311622 -> n172311621[govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="80" govpc_dstPort="0" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=80\ndstPort=0\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
					n1723116139 [label="172.31.16.139"]
					n172311621 [label="172.31.16.21"]
					n172311622 [label="172.31.16.22"]
				}`,
		},
		{
			Name: "added_port",
			Previous: `digraph {
					n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="20" govpc_bytes="1000" govpc_start="1418530010" govpc_end="1818530070" color=red label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=20\nbytes=1000\nstart=1418530010\nend=1818530070"]
					n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
					n172311621 -> n1723116139 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="80" govpc_dstPort="0" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=80\ndstPort=0\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
					n1723116139 [label="172.31.16.139"]
					n172311621 [label="172.31.16.21"]
				}`,
			Next: `digraph {
					n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="20" govpc_bytes="1000" govpc_start="1418530010" govpc_end="1818530070" color=red label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=20\nbytes=1000\nstart=1418530010\nend=1818530070"]
					n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
					n172311621 -> n1723116139 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="80" govpc_dstPort="0" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=80\ndstPort=0\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
					n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="22" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
					n172311621 -> n1723116139 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="22" govpc_dstPort="0" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=80\ndstPort=0\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
					n1723116139 [label="172.31.16.139"]
					n172311621 [label="172.31.16.21"]
				}`,
		},
		{
			Name: "removed_node",
			Next: `digraph {
					n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="20" govpc_bytes="1000" govpc_start="1418530010" govpc_end="1818530070" color=red label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=20\nbytes=1000\nstart=1418530010\nend=1818530070"]
					n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
					n172311621 -> n1723116139 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="80" govpc_dstPort="0" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=80\ndstPort=0\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
					n1723116139 [label="172.31.16.139"]
					n172311621 [label="172.31.16.21"]
				}`,
			Previous: `digraph {
					n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="20" govpc_bytes="1000" govpc_start="1418530010" govpc_end="1818530070" color=red label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=20\nbytes=1000\nstart=1418530010\nend=1818530070"]
					n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
					n172311621 -> n1723116139 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="80" govpc_dstPort="0" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=80\ndstPort=0\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
					n172311621 -> n172311622 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=80\ndstPort=0\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
					n172311622 -> n172311621[govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="80" govpc_dstPort="0" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=80\ndstPort=0\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
					n1723116139 [label="172.31.16.139"]
					n172311621 [label="172.31.16.21"]
					n172311622 [label="172.31.16.22"]
				}`,
		},
		{
			Name: "removed_port",
			Next: `digraph {
					n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="20" govpc_bytes="1000" govpc_start="1418530010" govpc_end="1818530070" color=red label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=20\nbytes=1000\nstart=1418530010\nend=1818530070"]
					n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
					n172311621 -> n1723116139 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="80" govpc_dstPort="0" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=80\ndstPort=0\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
					n1723116139 [label="172.31.16.139"]
					n172311621 [label="172.31.16.21"]
				}`,
			Previous: `digraph {
					n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="20" govpc_bytes="1000" govpc_start="1418530010" govpc_end="1818530070" color=red label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=20\nbytes=1000\nstart=1418530010\nend=1818530070"]
					n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
					n172311621 -> n1723116139 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="80" govpc_dstPort="0" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=80\ndstPort=0\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
					n1723116139 -> n172311621 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="22" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
					n172311621 -> n1723116139 [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="22" govpc_dstPort="0" govpc_protocol="6" govpc_packets="40" govpc_bytes="2000" govpc_start="1418530010" govpc_end="1818530070" color=green label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=80\ndstPort=0\nprotocol=6\npackets=40\nbytes=2000\nstart=1418530010\nend=1818530070"]
					n1723116139 [label="172.31.16.139"]
					n172311621 [label="172.31.16.21"]
				}`,
		},
	}
	d := domain.Diff{
		PreviousStart: time.Now().Add(-1 * time.Hour),
		PreviousStop:  time.Now().Add(-1 * time.Hour),
		NextStart:     time.Now(),
		NextStop:      time.Now(),
	}

	for _, tt := range tc {
		b.Run(tt.Name, func(b *testing.B) {
			ctrl := gomock.NewController(b)
			defer ctrl.Finish()

			grapherMock := NewMockGrapher(ctrl)

			for n := 0; n < b.N; n = n + 1 {
				grapherMock.EXPECT().Graph(gomock.Any(), d.PreviousStart, d.PreviousStop).Return(ioutil.NopCloser(bytes.NewReader([]byte(tt.Previous))), nil)
				grapherMock.EXPECT().Graph(gomock.Any(), d.PreviousStart, d.PreviousStop).Return(ioutil.NopCloser(bytes.NewReader([]byte(tt.Previous))), nil)
				grapherMock.EXPECT().Graph(gomock.Any(), d.NextStart, d.NextStop).Return(ioutil.NopCloser(bytes.NewReader([]byte(tt.Next))), nil)
				grapherMock.EXPECT().Graph(gomock.Any(), d.NextStart, d.NextStop).Return(ioutil.NopCloser(bytes.NewReader([]byte(tt.Next))), nil)
			}

			differ := DOTDiffer{grapherMock}

			b.ResetTimer()
			for n := 0; n < b.N; n = n + 1 {
				_, err := differ.Diff(context.Background(), d)
				if err != nil {
					b.Fatal(err.Error())
				}
			}
		})
	}
}

func BenchmarkLargeDiff(b *testing.B) {

	prevBuff := bytes.NewBufferString("digraph {\n")
	nextBuff := bytes.NewBufferString("digraph {\n")
	for z := 0; z < 2; z = z + 1 {
		for x := 0; x < 256; x = x + 1 {
			xIP := fmt.Sprintf("127.0.%d.%d", z, x)
			xID := "n" + strings.Replace(xIP, ".", "", -1)
			for y := 0; y < 256; y = y + 1 {
				yIP := fmt.Sprintf("127.0.%d.%d", z, y)
				yID := "n" + strings.Replace(yIP, ".", "", -1)
				_, _ = prevBuff.WriteString(fmt.Sprintf(`%s -> %s [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="20" govpc_bytes="1000" govpc_start="1418530010" govpc_end="1818530070" color=red label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=20\nbytes=1000\nstart=1418530010\nend=1818530070"]
			`, xID, yID))
				_, _ = prevBuff.WriteString(fmt.Sprintf(`%s [label="%s"]
			`, yID, yIP))
			}
			_, _ = prevBuff.WriteString(fmt.Sprintf(`%s [label="%s"]
		`, xID, xIP))
		}
		for x := 0; x < 256; x = x + 1 {
			xIP := fmt.Sprintf("127.1.%d.%d", z, x)
			xID := "n" + strings.Replace(xIP, ".", "", -1)
			for y := 0; y < 256; y = y + 1 {
				yIP := fmt.Sprintf("127.1.%d.%d", z, y)
				yID := "n" + strings.Replace(yIP, ".", "", -1)
				_, _ = nextBuff.WriteString(fmt.Sprintf(`%s -> %s [govpc_accountID="123456789010" govpc_eniID="eni-abc123de" govpc_srcPort="0" govpc_dstPort="80" govpc_protocol="6" govpc_packets="20" govpc_bytes="1000" govpc_start="1418530010" govpc_end="1818530070" color=red label="accountID=123456789010\neniID=eni-abc123de\nsrcPort=0\ndstPort=80\nprotocol=6\npackets=20\nbytes=1000\nstart=1418530010\nend=1818530070"]
			`, xID, yID))
				_, _ = nextBuff.WriteString(fmt.Sprintf(`%s [label="%s"]
			`, yID, yIP))
			}
			_, _ = nextBuff.WriteString(fmt.Sprintf(`%s [label="%s"]
		`, xID, xIP))
		}
	}
	_, _ = prevBuff.WriteString("}")
	_, _ = nextBuff.WriteString("}")
	d := domain.Diff{
		PreviousStart: time.Now().Add(-1 * time.Hour),
		PreviousStop:  time.Now().Add(-1 * time.Hour),
		NextStart:     time.Now(),
		NextStop:      time.Now(),
	}

	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	grapherMock := NewMockGrapher(ctrl)

	for n := 0; n < b.N; n = n + 1 {
		grapherMock.EXPECT().Graph(gomock.Any(), d.PreviousStart, d.PreviousStop).Return(ioutil.NopCloser(bytes.NewReader(prevBuff.Bytes())), nil)
		grapherMock.EXPECT().Graph(gomock.Any(), d.PreviousStart, d.PreviousStop).Return(ioutil.NopCloser(bytes.NewReader(prevBuff.Bytes())), nil)
		grapherMock.EXPECT().Graph(gomock.Any(), d.NextStart, d.NextStop).Return(ioutil.NopCloser(bytes.NewReader(nextBuff.Bytes())), nil)
		grapherMock.EXPECT().Graph(gomock.Any(), d.NextStart, d.NextStop).Return(ioutil.NopCloser(bytes.NewReader(nextBuff.Bytes())), nil)
	}

	differ := DOTDiffer{grapherMock}

	b.ResetTimer()
	for n := 0; n < b.N; n = n + 1 {
		_, err := differ.Diff(context.Background(), d)
		if err != nil {
			b.Fatal(err.Error())
		}
	}
}
