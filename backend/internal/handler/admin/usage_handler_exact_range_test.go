package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAdminUsageListFromToOverrideDates(t *testing.T) {
	repo := &adminUsageRepoCapture{}
	router := newAdminUsageRequestTypeTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet,
		"/admin/usage?start_date=2026-10-01&end_date=2026-10-31&timezone=UTC&from=2026-10-12T12:30:00Z&to=2026-10-12T13:30:00Z", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, repo.listFilters.StartTime)
	require.NotNil(t, repo.listFilters.EndTime)
	require.True(t, repo.listFilters.StartTime.Equal(time.Date(2026, 10, 12, 12, 30, 0, 0, time.UTC)))
	require.True(t, repo.listFilters.EndTime.Equal(time.Date(2026, 10, 12, 13, 30, 0, 0, time.UTC)))
}

func TestAdminUsageStatsFromToOverrideDates(t *testing.T) {
	repo := &adminUsageRepoCapture{}
	router := newAdminUsageRequestTypeTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet,
		"/admin/usage/stats?start_date=2026-10-01&end_date=2026-10-31&timezone=UTC&from=2026-10-12T15:30:00%2B03:00&to=2026-10-12T13:30:00Z", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, repo.statsFilters.StartTime)
	require.NotNil(t, repo.statsFilters.EndTime)
	require.True(t, repo.statsFilters.StartTime.Equal(time.Date(2026, 10, 12, 12, 30, 0, 0, time.UTC)))
	require.True(t, repo.statsFilters.EndTime.Equal(time.Date(2026, 10, 12, 13, 30, 0, 0, time.UTC)))
}
