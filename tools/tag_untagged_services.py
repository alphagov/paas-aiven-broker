import os
import logging
import requests
from requests.models import Response


LOGLEVEL = os.getenv("LOGLEVEL", "INFO")

logging.basicConfig(
    format="%(asctime)s %(levelname)s - %(message)s",
    datefmt="%Y-%m-%dT%H:%M:%S%z",
    level=LOGLEVEL,
)


class AivenClient:
    class AivenException(Exception):
        pass

    API_BASE = "https://api.aiven.io/v1"

    def __init__(self, token: str, project: str) -> None:
        self.api_token = token
        self.project = project
        self.request_headers = {"authorization": f"aivenv1 {self.api_token}"}
        self.baseurl = f"{self.API_BASE}/project/{self.project}"

    def get(self, path) -> Response:
        logging.debug(f"AivenClient:GET {path}")
        r = requests.get(
            f"{self.baseurl}{path}",
            headers=self.request_headers,
        )
        match r.status_code:
            case 401:
                raise AivenClient.AivenException(
                    "Aiven Authorization failed. Check AIVEN_API_TOKEN envar"
                )
            case 404:
                raise AivenClient.AivenException(
                    f"Aiven returned 404: {r.json()['message']}"
                )
            case _:
                return r

    def get_all_services(self) -> dict:
        r = self.get("/service")
        return r.json()["services"]

    def service_tags(self, service_name: str) -> bool:
        tags = self.get(f"/service/{service_name}/tags").json()["tags"]
        return tags


class CloudFoundryClient:
    def __init__(self, url: str, auth: str) -> None:
        self.baseurl = url
        self.auth_header = {"authorization": auth}

    def get(self, path):
        return requests.get(f"{self.baseurl}{path}", headers=self.auth_header)

    def get_service_space(self, service_id):
        r = self.get(f"/v3/service_instances/{service_id}").json()
        return r["relationships"]["space"]["data"]["guid"]

    def get_service_plan(self, service_id):
        r = self.get(f"/v3/service_instances/{service_id}").json()
        plan_guid = r["relationships"]["service_plan"]["data"]["guid"]
        r = self.get(f"/v3/service_plans/{plan_guid}").json()
        return r["broker_catalog"]["id"]

    def get_space_org(self, space_id):
        r = self.get(f"/v3/spaces/{space_id}").json()
        return r["relationships"]["organization"]["data"]["guid"]


def generate_basic_tags(cf_client: CloudFoundryClient, service_name: str) -> dict:
    tags = {
        "broker_name": "",
        "deploy_env": "",
        "organization_id": "",
        "plan_id": "",
        "restored_from_backup": "false",
        "restored_from_service": "",
        "restored_from_time": "0001-01-01T00:00:00Z",
        "service_id": "",
        "space_id": "",
    }
    tags["broker_name"] = tags["deploy_env"] = service_name.split("-")[0]
    tags["service_id"] = "-".join(service_name.split("-")[1:])

    tags["space_id"] = cf_client.get_service_space(tags["service_id"])
    tags["plan_id"] = cf_client.get_service_plan(tags["service_id"])
    tags["organization_id"] = cf_client.get_space_org(tags["space_id"])

    return tags


def main():
    aiven_token = os.getenv("AIVEN_API_TOKEN")
    aiven_project = os.getenv("AIVEN_PROJECT")
    aiven_client = AivenClient(aiven_token, aiven_project)
    cf_auth = os.getenv("CF_AUTH")
    cf_api_baseurl = os.getenv("CF_API_BASEURL")
    cf_client = CloudFoundryClient(cf_api_baseurl, cf_auth)

    logging.debug(f"getting services for project {aiven_project}")
    try:
        services = aiven_client.get_all_services()
    except Exception as e:
        logging.exception(e)
        raise
    logging.info(f"{len(services)} services retrieved from Aiven")

    for service in services:
        service_name = service["service_name"]
        service_tags = aiven_client.service_tags(service_name)
        tags_set = len(service_tags) != 0
        logging.debug(f"Service: {service_name} has tags: {tags_set}")
        if tags_set:
            continue
        new_tags = generate_basic_tags(cf_client, service_name)
        logging.debug(f"{service_name} -> old tags: {service_tags}")
        logging.debug(f"{service_name} -> new tags: {new_tags}")


if __name__ == "__main__":
    main()
