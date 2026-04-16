package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// ParseLocationFilter extracts location IDs from query params.
// Supports both ?location_id=single and ?location_ids=id1,id2,id3
func ParseLocationFilter(r *http.Request) []uuid.UUID {
	if ids := r.URL.Query().Get("location_ids"); ids != "" {
		parts := strings.Split(ids, ",")
		var result []uuid.UUID
		for _, p := range parts {
			if id, err := uuid.Parse(strings.TrimSpace(p)); err == nil {
				result = append(result, id)
			}
		}
		return result
	}
	if id := r.URL.Query().Get("location_id"); id != "" {
		if uid, err := uuid.Parse(id); err == nil {
			return []uuid.UUID{uid}
		}
	}
	return nil
}

// AddLocationFilter appends an "AND location_id = ANY($N)" clause when IDs are present.
func AddLocationFilter(args []interface{}, locationIDs []uuid.UUID) (string, []interface{}) {
	if len(locationIDs) == 0 {
		return "", args
	}
	idx := len(args) + 1
	return " AND location_id = ANY($" + strconv.Itoa(idx) + "::uuid[])", append(args, locationIDs)
}
