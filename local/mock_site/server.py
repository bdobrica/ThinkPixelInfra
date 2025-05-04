import json

import requests
from flask import Flask, Response, abort, request

app = Flask(__name__)

# module‐level “current” token
validation_token = None


@app.route("/", methods=["GET", "POST"])
def root():
    if request.args.get("rest_route") not in {
        "/thinkpixel/v1/validate/",
        "/thinkpixel/v1/exchange/",
    }:
        abort(404)

    # GET (and HEAD via Flask) → serve your stub
    if request.method == "GET":
        payload = {
            "domain": "example.com",
            "path": "/",
            "validation_token": validation_token,
            "nonce": "1234567890abcdef",
        }
        return Response(json.dumps(payload), mimetype="application/json")

    # POST → validate and echo
    data = request.get_json(silent=True)
    if data is None:
        abort(400, description="Invalid JSON")

    # log the API key for debug
    app.logger.info("API Key: %s", data.get("api_key"))

    return Response(
        json.dumps({"success": True, "message": "Validation token accepted"}),
        mimetype="application/json",
    )


@app.route("/register", methods=["GET"])
def register():
    global validation_token  # ← make sure we update the module‐level var

    resp = requests.post(
        "http://api-gateway:8080/register",
        json={
            "domain": "example.com",
            "path": "/",
            "estimated_pages": 415,
            "average_page_size": 1478,
            "st_dev_page_size": 4891,
        },
    )
    app.logger.info("Response from API Gateway: %s", resp.text)

    if resp.status_code != 200:
        abort(500, description="Failed to register with API Gateway")

    parsed = resp.json()
    validation_token = parsed.get("validation_token")
    if not validation_token:
        abort(500, description="Invalid response from API Gateway")

    return Response(
        json.dumps({"validation_token": validation_token}),
        mimetype="application/json",
    )


if __name__ == "__main__":
    # debug=True is fine for local testing; remove in prod
    app.run(host="0.0.0.0", port=80, debug=True)
