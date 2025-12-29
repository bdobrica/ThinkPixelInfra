import json
import os

import mysql.connector
from flask import Flask, Response, abort, request

app = Flask(__name__)

# Database connection configuration
DB_CONFIG = {
    "host": os.environ.get("DB_HOST", "mysql"),
    "user": os.environ.get("DB_USER", "root"),
    "password": os.environ.get("DB_PASSWORD", "root"),
    "database": os.environ.get("DB_NAME", "thinkpixel"),
}


def get_db_connection():
    """Create and return a database connection"""
    return mysql.connector.connect(**DB_CONFIG)


@app.route("/", methods=["GET", "POST"])
def root():
    if request.args.get("rest_route") not in {
        "/thinkpixel/v1/validate/",
        "/thinkpixel/v1/exchange/",
    }:
        abort(404)

    # GET (and HEAD via Flask) - serve your stub
    if request.method == "GET":
        # Query database for the latest validation token for this domain
        try:
            conn = get_db_connection()
            cursor = conn.cursor(dictionary=True)
            cursor.execute(
                "SELECT validation_token FROM wp_thinkpixel_sites"
                " WHERE domain='mock-site' AND path='/' AND validation_status='pending'"
                " ORDER BY created_at DESC LIMIT 1"
            )
            result = cursor.fetchone()
            cursor.close()
            conn.close()

            validation_token = result["validation_token"] if result else None
        except Exception as e:
            app.logger.error(f"Failed to query validation token: {e}")
            validation_token = None

        payload = {
            "domain": "mock-site",
            "path": "/",
            "validation_token": validation_token,
            "nonce": "1234567890abcdef",
        }
        return Response(json.dumps(payload), mimetype="application/json")

    # POST - validate and echo
    data = request.get_json(silent=True)
    if data is None:
        abort(400, description="Invalid JSON")

    # Store the API key for testing purposes
    api_key = data.get("api_key")
    if api_key:
        with open("/tmp/api_key.txt", "w") as f:
            f.write(api_key)
        app.logger.info("API Key stored: %s", api_key)

    return Response(
        json.dumps({"success": "true", "message": "Validation token accepted"}),
        mimetype="application/json",
    )


if __name__ == "__main__":
    # debug=True is fine for local testing; remove in prod
    app.run(host="0.0.0.0", port=80, debug=True)
