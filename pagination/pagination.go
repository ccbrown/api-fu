package pagination

import (
	"sort"
	"time"
)

// PageInfo represents the information for the current page of results.
type PageInfo[C Cursor[C]] struct {
	HasPreviousPage bool
	HasNextPage     bool
	StartCursor     *C
	EndCursor       *C
}

type Cursor[T any] interface {
	LessThan(T) bool
}

type Edge[C Cursor[C]] interface {
	Cursor() C
}

// Returns a new slice containing only the edges that are within the range specified by the given cursors.
func ApplyCursorsToEdges[E Edge[C], C Cursor[C]](edges []E, after, before *C) (filtered []E, hadEdgesBeforeAfter, hadEdgesAfterBefore bool) {
	if after == nil && before == nil {
		filtered = append([]E(nil), edges...)
	} else {
		for _, edge := range edges {
			c := edge.Cursor()
			if before != nil && !c.LessThan(*before) {
				hadEdgesAfterBefore = true
				continue
			}
			if after != nil && !(*after).LessThan(c) {
				hadEdgesBeforeAfter = true
				continue
			}
			filtered = append(filtered, edge)
		}
	}

	return filtered, hadEdgesBeforeAfter, hadEdgesAfterBefore
}

// Returns the page of edges that should be returned for the given pagination parameters.
func EdgesToReturn[E Edge[C], C Cursor[C]](edges []E, after, before *C, first, last *int) ([]E, PageInfo[C]) {
	var pageInfo PageInfo[C]
	edges, pageInfo.HasPreviousPage, pageInfo.HasNextPage = ApplyCursorsToEdges(edges, after, before)

	sort.Slice(edges, func(i, j int) bool {
		return edges[i].Cursor().LessThan(edges[j].Cursor())
	})

	if first != nil {
		if len(edges) > *first {
			edges = edges[:*first]
			pageInfo.HasNextPage = true
		} else {
			pageInfo.HasNextPage = false
		}
	}

	if last != nil {
		if len(edges) > *last {
			edges = edges[len(edges)-*last:]
			pageInfo.HasPreviousPage = true
		} else {
			pageInfo.HasPreviousPage = false
		}
	}

	if len(edges) > 0 {
		startCursor := edges[0].Cursor()
		pageInfo.StartCursor = &startCursor
		endCursor := edges[len(edges)-1].Cursor()
		pageInfo.EndCursor = &endCursor
	}

	return edges, pageInfo
}

type TimeBasedCursor[T any] interface {
	Cursor[T]
	Time() time.Time
}

type TimeBasedRangeQuery struct {
	MinTime time.Time
	MaxTime time.Time
	Limit   int
}

var distantFuture = time.Date(3000, time.January, 1, 0, 0, 0, 0, time.UTC)

// For a time-based request with the given parameters, returns a list of range queries that should
// be made to the resolver.
//
// Limit is the maximum number of items to return. If it is negative, the last `limit` items will be
// returned. If it is zero, there is no limit.
func TimeBasedRangeQueries[C TimeBasedCursor[C]](after, before *C, atOrAfterTimeIn, beforeTimeIn *time.Time, limit int) []TimeBasedRangeQuery {
	var queries []TimeBasedRangeQuery

	atOrAfterTime := time.Time{}
	if atOrAfterTimeIn != nil {
		atOrAfterTime = *atOrAfterTimeIn
	}

	beforeTime := distantFuture
	if beforeTimeIn != nil {
		beforeTime = *beforeTimeIn
	}

	middle := TimeBasedRangeQuery{atOrAfterTime, beforeTime.Add(-time.Nanosecond), limit}

	if after != nil {
		afterTime := (*after).Time()
		queries = append(queries, TimeBasedRangeQuery{afterTime, afterTime, 0})
		if t := time.Unix(0, afterTime.UnixNano()+1); t.After(middle.MinTime) {
			middle.MinTime = t
		}
	}

	if before != nil {
		beforeTime := (*before).Time()
		if after == nil || !(*after).Time().Equal(beforeTime) {
			queries = append(queries, TimeBasedRangeQuery{beforeTime, beforeTime, 0})
		}
		if t := time.Unix(0, beforeTime.UnixNano()-1); t.Before(middle.MaxTime) {
			middle.MaxTime = t
		}
	}

	queries = append(queries, middle)

	return queries
}
