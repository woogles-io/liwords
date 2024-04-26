Previously located at https://github.com/domino14/tourneypdf

A Python service to create PDF scoresheets for Woogles tourneys, with optional QR codes.

This should be implemented as a serverless/lambda function on production.

### Deployment

Manual for now

#### If we change any dependencies
1) docker build -t pycairo-layer .
2) docker run --rm -v $(pwd)/output:/output pycairo-layer
3) upload the layer with AWS console (to AWS Lambda Layers)
4) Update the template.yaml with the new layer ARN

#### Regular deploy:
5) sam build -t template.yaml --use-container --build-image TourneyPDFFunction=pycairo-layer
6) AWS_PROFILE=prod sam deploy --guided  (use all defaults)