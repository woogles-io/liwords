Previously located at https://github.com/domino14/tourneypdf

A Python service to create PDF scoresheets for Woogles tourneys, with optional QR codes.

This should be implemented as a serverless/lambda function on production.

### Deployment

Manual for now

1) docker build -t my-custom-lambda-build-image .
2) sam build -t template.yaml --use-container --build-image TourneyPDFFunction=my-custom-lambda-build-image
3) AWS_PROFILE=prod sam deploy --guided  (use all defaults)