package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSpecificLogs(t *testing.T) {
	t.Skip("Skipping test because it is specific to Pavel's system")
	settings := NewSettings("/Users/pavel/Work/2022-12-07-nautobot/dolt-nautobot/test_data/logs-PlatformTestCase-march-23/dolt-sql.log", "")
	result, err := mainLogic(settings)
	require.NoError(t, err)
	require.Equal(t, "/Users/pavel/Work/2022-12-07-nautobot/dolt-nautobot/test_data/logs-PlatformTestCase-march-23/dolt-sql.queries.log", result.queriesOutputPath)
}

func TestLatestLogs(t *testing.T) {
	t.Skip("Skipping test because it is specific to Pavel's system")
	settings := NewSettings(
		"/Users/pavel/Work/2022-12-07-nautobot/dolt-nautobot/test_data/logs/dolt-sql.log",
		"/Users/pavel/Work/2022-12-07-nautobot/dolt-nautobot/test_data/logs/dolt-run.log")
	settings.logger = NewNoopLogger()
	result, err := mainLogic(settings)
	require.NoError(t, err)
	require.Equal(t,
		"/Users/pavel/Work/2022-12-07-nautobot/dolt-nautobot/test_data/logs/dolt-sql.queries.log",
		result.queriesOutputPath)
}

func TestPlatformTestCase(t *testing.T) {
	settings := NewSettings(
		"test-logs/nautobot.dcim.tests.test_filters.PlatformTestCase.sql.txt",
		"test-logs/nautobot.dcim.tests.test_filters.PlatformTestCase.pytest.txt")
	result, err := mainLogic(settings)
	require.NoError(t, err)
	require.Equal(t, "test-logs/nautobot.dcim.tests.test_filters.PlatformTestCase.sql.queries.txt", result.queriesOutputPath)
	require.Equal(t, "test-logs/nautobot.dcim.tests.test_filters.PlatformTestCase.sql.analysis.txt", result.analysisOutputPath)
	require.Equal(t, "test-logs/nautobot.dcim.tests.test_filters.PlatformTestCase.sql.patch_queries.txt", result.patchQueriesOutputPath)
}

func TestSampleCase(t *testing.T) {
	// prepare
	logs := []string{
		"2023-02-09T18:30:09Z DEBUG [conn 3] Query finished in 12 ms {connectTime=2023-02-09T18:30:05Z, connectionDb=test_nautobot, query=SELECT `extras_job`.`id`, `extras_job`.`created`, `extras_job`.`last_updated`, `extras_job`.`_custom_field_data`, `extras_job`.`source`, `extras_job`.`module_name`, `extras_job`.`job_class_name`, `extras_job`.`slug`, `extras_job`.`grouping`, `extras_job`.`name`, `extras_job`.`description`, `extras_job`.`installed`, `extras_job`.`enabled`, `extras_job`.`commit_default`, `extras_job`.`hidden`, `extras_job`.`read_only`, `extras_job`.`approval_required`, `extras_job`.`soft_time_limit`, `extras_job`.`time_limit`, `extras_job`.`grouping_override`, `extras_job`.`name_override`, `extras_job`.`description_override`, `extras_job`.`commit_default_override`, `extras_job`.`hidden_override`, `extras_job`.`read_only_override`, `extras_job`.`approval_required_override`, `extras_job`.`soft_time_limit_override`, `extras_job`.`time_limit_override`, `extras_job`.`git_repository_id`, `extras_job`.`has_sensitive_variables`, `extras_job`.`has_sensitive_variables_override`, `extras_job`.`is_job_hook_receiver`, `extras_job`.`task_queues`, `extras_job`.`task_queues_override` FROM `extras_job` WHERE (`extras_job`.`git_repository_id` IS NULL AND `extras_job`.`job_class_name` = 'TestFileUploadPass' AND `extras_job`.`module_name` = 'test_file_upload_pass' AND `extras_job`.`source` = 'local') LIMIT 21}",
		"2023-03-05T00:13:41Z WARN [conn 329] error running query {connectTime=2023-03-05T00:13:13Z, connectionDb=test_nautobot, error=nil operand found in comparison, query=SELECT `ipam_prefix`.`id`, `ipam_prefix`.`created`, `ipam_prefix`.`last_updated`, `ipam_prefix`.`_custom_field_data`, `ipam_prefix`.`status_id`, `ipam_prefix`.`network`, `ipam_prefix`.`broadcast`, `ipam_prefix`.`prefix_length`, `ipam_prefix`.`site_id`, `ipam_prefix`.`location_id`, `ipam_prefix`.`vrf_id`, `ipam_prefix`.`tenant_id`, `ipam_prefix`.`vlan_id`, `ipam_prefix`.`role_id`, `ipam_prefix`.`is_pool`, `ipam_prefix`.`description` FROM `ipam_prefix` LEFT OUTER JOIN `ipam_vrf` ON (`ipam_prefix`.`vrf_id` = `ipam_vrf`.`id`) WHERE `ipam_prefix`.`prefix_length` = 52 ORDER BY `ipam_vrf`.`name` ASC, `ipam_prefix`.`network` ASC, `ipam_prefix`.`prefix_length` ASC}",
		"2023-03-22T21:54:58Z DEBUG [conn 3] Starting query {connectTime=2023-03-22T21:54:44Z, connectionDb=test_nautobot, query=U0VMRUNUIGBkamFuZ29fY29udGVudF90eXBlYC5gaWRgLCBgZGphbmdvX2NvbnRlbnRfdHlwZWAuYGFwcF9sYWJlbGAsIGBkamFuZ29fY29udGVudF90eXBlYC5gbW9kZWxgIEZST00gYGRqYW5nb19jb250ZW50X3R5cGVgIElOTkVSIEpPSU4gYGV4dHJhc190YWdfY29udGVudF90eXBlc2AgT04gKGBkamFuZ29fY29udGVudF90eXBlYC5gaWRgID0gYGV4dHJhc190YWdfY29udGVudF90eXBlc2AuYGNvbnRlbnR0eXBlX2lkYCkgV0hFUkUgYGV4dHJhc190YWdfY29udGVudF90eXBlc2AuYHRhZ19pZGAgPSAnY2NjYzIxNjQzNDNlNDUxZGIxMzhiN2ZkNTlmOWJjODYn}",
		"2023-03-22T21:54:43Z WARN [conn 1] error running query {connectTime=2023-03-22T21:54:43Z, connectionDb=nautobot, error=table not found: django_content_type, query=U0VMRUNUIGBkamFuZ29fY29udGVudF90eXBlYC5gaWRgLCBgZGphbmdvX2NvbnRlbnRfdHlwZWAuYGFwcF9sYWJlbGAsIGBkamFuZ29fY29udGVudF90eXBlYC5gbW9kZWxgIEZST00gYGRqYW5nb19jb250ZW50X3R5cGVgIFdIRVJFICgoYGRqYW5nb19jb250ZW50X3R5cGVgLmBhcHBfbGFiZWxgID0gJ2V4dHJhcycgQU5EIGBkamFuZ29fY29udGVudF90eXBlYC5gbW9kZWxgIElOICgncmVsYXRpb25zaGlwYXNzb2NpYXRpb24nLCAnc3RhdHVzJywgJ3RhZycsICdkeW5hbWljZ3JvdXAnLCAnY29uZmlnY29udGV4dHNjaGVtYScsICdzZWNyZXQnLCAnc2VjcmV0c2dyb3VwJykpIE9SIChgZGphbmdvX2NvbnRlbnRfdHlwZWAuYGFwcF9sYWJlbGAgPSAnZGNpbScgQU5EIGBkamFuZ29fY29udGVudF90eXBlYC5gbW9kZWxgIElOICgnY29uc29sZXBvcnQnLCAnY29uc29sZXNlcnZlcnBvcnQnLCAncG93ZXJwb3J0JywgJ3Bvd2Vyb3V0bGV0JywgJ2ludGVyZmFjZScsICdmcm9udHBvcnQnLCAncmVhcnBvcnQnLCAnZGV2aWNlYmF5JywgJ2ludmVudG9yeWl0ZW0nLCAnbWFudWZhY3R1cmVyJywgJ2RldmljZXR5cGUnLCAnZGV2aWNlcm9sZScsICdwbGF0Zm9ybScsICdkZXZpY2UnLCAndmlydHVhbGNoYXNzaXMnLCAnZGV2aWNlcmVkdW5kYW5jeWdyb3VwJywgJ2NhYmxlJywgJ2NvbnNvbGVwb3J0dGVtcGxhdGUnLCAnY29uc29sZXNlcnZlcnBvcnR0ZW1wbGF0ZScsICdwb3dlcnBvcnR0ZW1wbGF0ZScsICdwb3dlcm91dGxldHRlbXBsYXRlJywgJ2ludGVyZmFjZXRlbXBsYXRlJywgJ2Zyb250cG9ydHRlbXBsYXRlJywgJ3JlYXJwb3J0dGVtcGxhdGUnLCAnZGV2aWNlYmF5dGVtcGxhdGUnLCAnbG9jYXRpb250eXBlJywgJ2xvY2F0aW9uJywgJ3Bvd2VycGFuZWwnLCAncG93ZXJmZWVkJywgJ3JhY2tncm91cCcsICdyYWNrcm9sZScsICdyYWNrJywgJ3JhY2tyZXNlcnZhdGlvbicsICdyZWdpb24nLCAnc2l0ZScpKSBPUiAoYGRqYW5nb19jb250ZW50X3R5cGVgLmBhcHBfbGFiZWxgID0gJ2NpcmN1aXRzJyBBTkQgYGRqYW5nb19jb250ZW50X3R5cGVgLmBtb2RlbGAgSU4gKCdwcm92aWRlcm5ldHdvcmsnLCAncHJvdmlkZXInLCAnY2lyY3VpdHR5cGUnLCAnY2lyY3VpdCcsICdjaXJjdWl0dGVybWluYXRpb24nKSkgT1IgKGBkamFuZ29fY29udGVudF90eXBlYC5gYXBwX2xhYmVsYCA9ICd2aXJ0dWFsaXphdGlvbicgQU5EIGBkamFuZ29fY29udGVudF90eXBlYC5gbW9kZWxgIElOICgnY2x1c3RlcnR5cGUnLCAnY2x1c3Rlcmdyb3VwJywgJ2NsdXN0ZXInLCAndmlydHVhbG1hY2hpbmUnLCAndm1pbnRlcmZhY2UnKSkgT1IgKGBkamFuZ29fY29udGVudF90eXBlYC5gYXBwX2xhYmVsYCA9ICdpcGFtJyBBTkQgYGRqYW5nb19jb250ZW50X3R5cGVgLmBtb2RlbGAgSU4gKCd2cmYnLCAncm91dGV0YXJnZXQnLCAncmlyJywgJ2FnZ3JlZ2F0ZScsICdyb2xlJywgJ3ByZWZpeCcsICdpcGFkZHJlc3MnLCAndmxhbmdyb3VwJywgJ3ZsYW4nLCAnc2VydmljZScpKSBPUiAoYGRqYW5nb19jb250ZW50X3R5cGVgLmBhcHBfbGFiZWxgID0gJ3RlbmFuY3knIEFORCBgZGphbmdvX2NvbnRlbnRfdHlwZWAuYG1vZGVsYCBJTiAoJ3RlbmFudGdyb3VwJywgJ3RlbmFudCcpKSBPUiAoYGRqYW5nb19jb250ZW50X3R5cGVgLmBhcHBfbGFiZWxgID0gJ2V4YW1wbGVfcGx1Z2luJyBBTkQgYGRqYW5nb19jb250ZW50X3R5cGVgLmBtb2RlbGAgSU4gKCdleGFtcGxlbW9kZWwnLCAnYW5vdGhlcmV4YW1wbGVtb2RlbCcpKSk=}",
		"2023-02-09T18:30:09Z DEBUG [conn 3] Query finished in 12 ms {connectTime=2023-02-09T18:30:05Z, connectionDb=test_nautobot, query=ROLLBACK TO SAVEPOINT s281473537629440_x46}",
	}
	input, err := os.CreateTemp("", "dolt-sql.log")
	require.NoError(t, err)
	defer os.Remove(input.Name())

	// Write the logs to the input file
	for _, log := range logs {
		_, err = input.WriteString(log + "\n")
		require.NoError(t, err)
	}
	err = input.Close()
	require.NoError(t, err)

	settings := NewSettings(input.Name(), "")
	settings.logger = NewTestLogger(t)

	// Run the main logic
	result, err := mainLogic(settings)
	require.NoError(t, err)

	// Read the queries output file
	outBytes, err := os.ReadFile(result.queriesOutputPath)
	require.NoError(t, err)
	outText := string(outBytes)
	require.NotEmpty(t, outText)

	require.Contains(t, outText, "Line 1")
	require.Contains(t, outText, "Line 2")
	require.NotContains(t, outText, "Line 3") // we only process "Query finished" lines
	require.Contains(t, outText, "Line 4")

	require.Contains(t, outText, "Query error: table not found: django_content_type")
	require.Contains(t, outText, "django_content_type.app_label")
	require.Contains(t, outText, "ipam_prefix.prefix_length ASC nullsFirst")
}
