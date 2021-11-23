// Code generated by smithy-go-codegen DO NOT EDIT.

package cloudwatchlogs

import (
	"context"
	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// Tests the filter pattern of a metric filter against a sample of log event
// messages. You can use this operation to validate the correctness of a metric
// filter pattern.
func (c *Client) TestMetricFilter(ctx context.Context, params *TestMetricFilterInput, optFns ...func(*Options)) (*TestMetricFilterOutput, error) {
	if params == nil {
		params = &TestMetricFilterInput{}
	}

	result, metadata, err := c.invokeOperation(ctx, "TestMetricFilter", params, optFns, c.addOperationTestMetricFilterMiddlewares)
	if err != nil {
		return nil, err
	}

	out := result.(*TestMetricFilterOutput)
	out.ResultMetadata = metadata
	return out, nil
}

type TestMetricFilterInput struct {

	// A symbolic description of how CloudWatch Logs should interpret the data in each
	// log event. For example, a log event can contain timestamps, IP addresses,
	// strings, and so on. You use the filter pattern to specify what to look for in
	// the log event message.
	//
	// This member is required.
	FilterPattern *string

	// The log event messages to test.
	//
	// This member is required.
	LogEventMessages []string

	noSmithyDocumentSerde
}

type TestMetricFilterOutput struct {

	// The matched events.
	Matches []types.MetricFilterMatchRecord

	// Metadata pertaining to the operation's result.
	ResultMetadata middleware.Metadata

	noSmithyDocumentSerde
}

func (c *Client) addOperationTestMetricFilterMiddlewares(stack *middleware.Stack, options Options) (err error) {
	err = stack.Serialize.Add(&awsAwsjson11_serializeOpTestMetricFilter{}, middleware.After)
	if err != nil {
		return err
	}
	err = stack.Deserialize.Add(&awsAwsjson11_deserializeOpTestMetricFilter{}, middleware.After)
	if err != nil {
		return err
	}
	if err = addSetLoggerMiddleware(stack, options); err != nil {
		return err
	}
	if err = awsmiddleware.AddClientRequestIDMiddleware(stack); err != nil {
		return err
	}
	if err = smithyhttp.AddComputeContentLengthMiddleware(stack); err != nil {
		return err
	}
	if err = addResolveEndpointMiddleware(stack, options); err != nil {
		return err
	}
	if err = v4.AddComputePayloadSHA256Middleware(stack); err != nil {
		return err
	}
	if err = addRetryMiddlewares(stack, options); err != nil {
		return err
	}
	if err = addHTTPSignerV4Middleware(stack, options); err != nil {
		return err
	}
	if err = awsmiddleware.AddRawResponseToMetadata(stack); err != nil {
		return err
	}
	if err = awsmiddleware.AddRecordResponseTiming(stack); err != nil {
		return err
	}
	if err = addClientUserAgent(stack); err != nil {
		return err
	}
	if err = smithyhttp.AddErrorCloseResponseBodyMiddleware(stack); err != nil {
		return err
	}
	if err = smithyhttp.AddCloseResponseBodyMiddleware(stack); err != nil {
		return err
	}
	if err = addOpTestMetricFilterValidationMiddleware(stack); err != nil {
		return err
	}
	if err = stack.Initialize.Add(newServiceMetadataMiddleware_opTestMetricFilter(options.Region), middleware.Before); err != nil {
		return err
	}
	if err = addRequestIDRetrieverMiddleware(stack); err != nil {
		return err
	}
	if err = addResponseErrorMiddleware(stack); err != nil {
		return err
	}
	if err = addRequestResponseLogging(stack, options); err != nil {
		return err
	}
	return nil
}

func newServiceMetadataMiddleware_opTestMetricFilter(region string) *awsmiddleware.RegisterServiceMetadata {
	return &awsmiddleware.RegisterServiceMetadata{
		Region:        region,
		ServiceID:     ServiceID,
		SigningName:   "logs",
		OperationName: "TestMetricFilter",
	}
}