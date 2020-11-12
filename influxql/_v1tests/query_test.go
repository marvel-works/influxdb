package v1tests

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"
)

// Ensure parameterized queries can be executed
func TestServer_Query_Parameterized(t *testing.T) {
	t.Parallel()
	s := OpenServer(t)
	defer s.Close()

	writes := []string{
		fmt.Sprintf(`cpu,host=foo value=1.0 %d`, mustParseTime(time.RFC3339Nano, "2000-01-01T01:00:00Z").UnixNano()),
		fmt.Sprintf(`cpu,host=bar value=1.0 %d`, mustParseTime(time.RFC3339Nano, "2000-01-02T01:00:00Z").UnixNano()),
	}

	test := NewTest("db0", "rp0")
	test.writes = Writes{
		&Write{data: strings.Join(writes, "\n")},
	}

	minTime := mustParseTime(time.RFC3339Nano, "2000-01-01T01:00:00Z").UnixNano()
	maxTime := mustParseTime(time.RFC3339Nano, "2000-01-01T02:00:00Z").UnixNano()

	test.addQueries([]*Query{
		{
			name:    "parameterized time",
			params:  url.Values{"db": []string{"db0"}, "params": []string{fmt.Sprintf(`{"0": %d, "1": %d}`, minTime, maxTime)}},
			command: `SELECT value FROM cpu WHERE time >= $0 AND time < $1`,
			exp:     `{"results":[{"statement_id":0,"series":[{"name":"cpu","columns":["time","value"],"values":[["2000-01-01T01:00:00Z",1]]}]}]}`,
		},
		{
			name:    "parameterized tag",
			params:  url.Values{"db": []string{"db0"}, "params": []string{`{"0": "foo"}`}},
			command: `SELECT value FROM cpu WHERE host = $0`,
			exp:     `{"results":[{"statement_id":0,"series":[{"name":"cpu","columns":["time","value"],"values":[["2000-01-01T01:00:00Z",1]]}]}]}`,
		},
	}...)

	ctx := context.Background()
	test.Run(ctx, t, s)
}

// Ensure queries are properly chunked
func TestServer_Query_Chunked(t *testing.T) {
	t.Parallel()
	s := OpenServer(t)
	defer s.Close()

	writes := []string{
		fmt.Sprintf(`cpu,host=foo value=1.0 %d`, mustParseTime(time.RFC3339Nano, "2000-01-01T01:00:00Z").UnixNano()),
		fmt.Sprintf(`cpu,host=bar value=1.0 %d`, mustParseTime(time.RFC3339Nano, "2000-01-02T01:00:00Z").UnixNano()),
	}

	test := NewTest("db0", "rp0")
	test.writes = Writes{
		&Write{data: strings.Join(writes, "\n")},
	}

	test.addQueries([]*Query{
		{
			name:    "query is chunked",
			params:  url.Values{"db": []string{"db0"}, "chunked": []string{"true"}, "chunk_size": []string{"1"}},
			command: `SELECT value FROM cpu`,
			exp: `{"results":[{"statement_id":0,"series":[{"name":"cpu","columns":["time","value"],"values":[["2000-01-01T01:00:00Z",1]],"partial":true}],"partial":true}]}
{"results":[{"statement_id":0,"series":[{"name":"cpu","columns":["time","value"],"values":[["2000-01-02T01:00:00Z",1]]}]}]}`,
		},
		{
			name:    "query is not chunked",
			params:  url.Values{"db": []string{"db0"}, "chunked": []string{"false"}, "chunk_size": []string{"1"}},
			command: `SELECT value FROM cpu`,
			exp:     `{"results":[{"statement_id":0,"series":[{"name":"cpu","columns":["time","value"],"values":[["2000-01-01T01:00:00Z",1],["2000-01-02T01:00:00Z",1]]}]}]}`,
		},
	}...)

	ctx := context.Background()
	test.Run(ctx, t, s)
}
