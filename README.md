# go-zaloa

Serves Terrain tiles using the [Mapzen terrain tiles](https://registry.opendata.aws/terrain-tiles/) with more complex shapes. Zaloa supports 256x256, 512x512, 260x260, and 516x516 pixel tiles. The source tiles are 256x256 pixels, so Zaloa fetches multiple tiles and stitches them together to get the other tile sizes.   

This is a port of the Python [zaloa](https://github.com/tilezen/zaloa) to Go.
 
## Deploying as AWS Lambda

Zaloa can be deployed as a Lambda to AWS. To do this:

### Compile Binary

I'm running on an M1 Mac, so I need to install a cross compiler to generate the binary we'll send to AWS Lambda. This step is unnecessary if you can compile on an amd64 Linux machine:

```shell
brew install filosottile/musl-cross/musl-cross
```

Once your system is set up to compile to amd64+linux, compile the `cmd/lambda.go` file into a binary (the `CC`, `CXX`, and `CGO_LDFLAGS` might not be necessary if you're not on a M1 Mac):

```shell
CGO_ENABLED=1 \
GOOS=linux \
CC=x86_64-linux-musl-gcc \
CXX=x86_64-linux-musl-g++ \
CGO_LDFLAGS="-static" \
go build -o output cmd/lambda/main.go
```

You'll end up with a file called `output` that is the compiled Zaloa binary.

### Package the Binary

The AWS Lambda APIs expect the binary in a .zip file, so package it like so:

```shell
zip -j output output.zip
```

You should have a file called `output.zip` that is about half the size of your `output` binary.

### Create a Lambda Execution Role

Your Lambda will execute in AWS with a role. You use this role to specify what permissions it has. In this case, since the elevation tiles are public, our policy can be blank, but it still needs to exist. Create one using the policy JSON in this repository using the `aws` CLI:

```shell
aws iam create-role \
  --role-name lambda-function-executor \
  --assume-role-policy-document file://./trust-policy.json
```

Let's attach the built-in basic Lambda execution policy to this role you just created:

```shell
aws iam attach-role-policy \
  --role-name lambda-function-executor \
  --policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole
```

### Create the Lambda

Now that the role is created, you need to upload the binary and create the lambda.

**Note**: You must replace `arn:aws:iam::1234567:role/lambda-function-executor` in my example here with the ARN of the role (named `lambda-function-executor`) you created above.

```shell
AWS_DEFAULT_REGION=us-east-1 \
aws lambda create-function \
  --function-name zaloa \
  --runtime go1.x \
  --role="arn:aws:iam::1234567:role/lambda-function-executor" \
  --handler main \
  --zip-file fileb://./output.zip
```

### Generate a URL

Now that your Lambda exists in AWS, you need a way to execute it and get the results into your browser. You might consider [API Gateway](https://docs.aws.amazon.com/lambda/latest/dg/services-apigateway.html) if you need access control or tighter integration with custom domains and don't mind spending a little bit more per request. If you don't care about that because you're just using it for yourself, you can use a [Lambda Function URL](https://docs.aws.amazon.com/lambda/latest/dg/lambda-urls.html) instead.

You can create one with a single AWS CLI command:

```shell
aws lambda create-function-url-config \
  --function-name="zaloa" \
  --auth-type="NONE"
```

This will return a URL with an `on.aws` suffix. Put that in your browser with a `/tilezen/terrain/v2/512/terrarium/0/0/0.png` suffix and you should get an image displayed in your browser.

### Viewing Logs

You can view logs for your Lambda by logging into the AWS console, browsing to the CloudWatch Logs section, and opening the "zaloa" logging group.

### Updating Code

To update to a newer version of Zaloa, you need to rebuild the binary, zip it up again, and then update the code for your Lambda with the new zipped binary:

```shell
aws lambda update-function-code \
  --function-name zaloa \
  --zip-file fileb://./output.zip
```
