import os
import mock
import pytest


import tag_untagged_services

UNTAGGED_SERVICES = [{"service_name": "test-some-untagged-service-uuid"}]


def test_correct_tags_generated_with_mocks(requests_mock):

    expected_tags = {
        "broker_name": "test",
        "deploy_env": "test",
        "organization_id": "org-uuid",
        "plan_id": "plan-uuid",
        "restored_from_backup": "false",
        "restored_from_service": "",
        "restored_from_time": "0001-01-01T00:00:00Z",
        "service_id": "service-uuid",
        "space_id": "space-uuid",
    }
    cfapi = tag_untagged_services.CloudFoundryClient(
        "https://api.testinstance.null", "dummy-key"
    )

    requests_mock.get(
        f"{cfapi.baseurl}/v3/service_instances/{expected_tags['service_id']}",
        json={
            "relationships": {
                "space": {"data": {"guid": expected_tags["space_id"]}},
                "service_plan": {"data": {"guid": "annoying-other-plan-guid-what"}},
            }
        },
    )
    requests_mock.get(
        f"{cfapi.baseurl}/v3/service_plans/annoying-other-plan-guid-what",
        json={"broker_catalog": {"id": expected_tags["plan_id"]}},
    )
    requests_mock.get(
        f"{cfapi.baseurl}/v3/spaces/{expected_tags['space_id']}",
        json={
            "relationships": {
                "organization": {"data": {"guid": expected_tags["organization_id"]}}
            }
        },
    )
    assert (
        tag_untagged_services.generate_basic_tags(cfapi, "test-service-uuid")
        == expected_tags
    )


def test_correct_tags_generated_live():
    SERVICE_NAME = os.getenv("TEST_SERVICE_NAME")
    assert SERVICE_NAME is not None

    aiven_token = os.getenv("AIVEN_API_TOKEN")
    assert aiven_token is not None
    aiven_project = os.getenv("AIVEN_PROJECT")
    assert aiven_project is not None
    aiven_client = tag_untagged_services.AivenClient(aiven_token, aiven_project)

    cf_api_baseurl = os.getenv("CF_API_BASEURL")
    assert cf_api_baseurl is not None
    cf_auth = os.getenv("CF_AUTH")
    assert cf_auth is not None
    assert cf_auth.startswith("bearer ")
    cf_client = tag_untagged_services.CloudFoundryClient(cf_api_baseurl, cf_auth)

    correct_tags = aiven_client.service_tags(SERVICE_NAME)

    assert (
        tag_untagged_services.generate_basic_tags(cf_client, SERVICE_NAME)
        == correct_tags
    )
