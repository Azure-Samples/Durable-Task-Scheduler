import logging
import os
from azure.storage.blob import BlobServiceClient
import azure.functions as func
import azure.durable_functions as df
from azure.identity import DefaultAzureCredential
from azure.ai.formrecognizer import DocumentAnalysisClient
import json
import time
from requests import get, post
import requests
from datetime import datetime

my_app = df.DFApp(http_auth_level=func.AuthLevel.ANONYMOUS)

# Use managed identity to access blob storage
credential = DefaultAzureCredential()
storage_account_name = os.environ.get("STORAGE_ACCOUNT_NAME")
if not storage_account_name:
    raise ValueError("STORAGE_ACCOUNT_NAME environment variable is required but not set")

blob_service_client = BlobServiceClient(
    account_url=f"https://{storage_account_name}.blob.core.windows.net",
    credential=credential
)

@my_app.blob_trigger(arg_name="myblob", path="input", connection="AzureWebJobsStorage")
@my_app.durable_client_input(client_name="client")
async def blob_trigger(myblob: func.InputStream, client):
    logging.info(f"Python blob trigger function processed blob"
                f"Name: {myblob.name}"
                f"Blob Size: {myblob.length} bytes")

    blobName = myblob.name.split("/")[1]
    await client.start_new("process_document", client_input=blobName)

# Orchestrator
@my_app.orchestration_trigger(context_name="context")
def process_document(context):
    blobName: str = context.get_input()

    first_retry_interval_in_milliseconds = 5000
    max_number_of_attempts = 3
    retry_options = df.RetryOptions(first_retry_interval_in_milliseconds, max_number_of_attempts)

    # Download the PDF from Blob Storage and use Document Intelligence Form Recognizer to analyze its contents.
    result = yield context.call_activity_with_retry("analyze_pdf", retry_options, blobName)
    # Send the analyzed contents to Azure OpenAI to generate a summary.
    result2 = yield context.call_activity_with_retry("summarize_text",  retry_options, result)
    # Save the summary to a new file and upload it back to storage.
    result3 = yield context.call_activity_with_retry("write_doc", retry_options, { "blobName": blobName, "summary": result2 })

    return logging.info(f"Successfully uploaded summary to {result3}")

@my_app.activity_trigger(input_name='blobName')
def analyze_pdf(blobName):
    logging.info(f"in analyze_text activity")
    global blob_service_client
    container_client = blob_service_client.get_container_client("input")
    blob_client = container_client.get_blob_client(blobName)
    blob =  blob_client.download_blob().read()
    doc = ''

    endpoint = os.environ["COGNITIVE_SERVICES_ENDPOINT"]
    credential = DefaultAzureCredential()

    document_analysis_client = DocumentAnalysisClient(endpoint, credential)

    poller = document_analysis_client.begin_analyze_document("prebuilt-layout", document=blob, locale="en-US")
    result = poller.result().pages

    for page in result:
        for line in page.lines:
            doc += line.content

    return doc

@my_app.activity_trigger(input_name='results')
@my_app.generic_input_binding(arg_name="response", type="textCompletion", data_type=func.DataType.STRING, prompt="Can you explain what the following text is about? {results}", model = "%CHAT_MODEL_DEPLOYMENT_NAME%")
def summarize_text(results, response: str):
    logging.info(f"in summarize_text activity")
    response_json = json.loads(response)
    logging.info(response_json['content'])
    return response_json

@my_app.activity_trigger(input_name='results')
def write_doc(results):
    logging.info(f"in write_doc activity")
    global blob_service_client
    container_client=blob_service_client.get_container_client("output")

    summary = results['blobName'] + "-" + str(datetime.now())
    sanitizedSummary = summary.replace(".", "-")
    fileName = sanitizedSummary + ".txt"

    logging.info("uploading to blob" + results['summary']['content'])
    container_client.upload_blob(name=fileName, data=results['summary']['content'])
    return str(summary + ".txt")
