# Tinybatch
A configurable micro-batcher with generics support.

## How does it work?
Build your tinybatch client with your preferred configuration options, start the processor and start sending it jobs to process in batches.

## How to use
1. Define your Job and JobResult structs
```go
type Job struct {
	ID     string
	Params struct {
		Name string
	}
}

type JobResult struct {
	ID     string
	Message string
	Error  string
}
```
2. Create the processBatch function
The function should take a slice of your job struct and return a slice of JobResult, along with an error
```go
func processJobs(jobs []Job) ([]JobResult, error) {

	results := make([]JobResult, 0, len(jobs))

	for _, job := range jobs {
		result := JobResult{
			ID:     job.ID,
			Message: job.Params.Name + " processed",
			Error:  "",
		}
		successCounter.Add(1)

		results = append(results, result)
	}

	return results, nil
}
```
3. Create a new batcher

```go
batcher := New(
	ctx, // required
	processJobs, // required
	WithMaxBatchSize(100), //optional - default: 100 
	WithMaxBatchDelay(1_000 * time.Millisecond), // optional - default: 1_000 * time.Millisecond
)
```
4. Start processing
```go
batcher.Start()
```

5. Start adding items to the batch
The add function will return an error if you try add something after context has been cancelled
```go
for i := 0; i < count; i++ {
	err := batcher.Add(Job{
		ID: strconv.Itoa(i),
		Params: struct {
			Name string
		}{
			Name: "Job " + strconv.Itoa(i),
		},
	})

	if err != nil {
		panic(err)
	}
}
```

## Makefile
### Vet
Runs Go vet
Runs the linter
```make vet```

### Test
Runs the package tests
```make test```


### Test CI
Runs the package tests without outputting test coverage
```make test_ci```

### test-coverage
Opens the test coverage report in a HTML file
```make test-coverage```

## Use cases
### Event stream to Firehose
Let's say you want to stream events into an S3 bucket. Writing each message to S3 individually would be very expensive and you would likely hit rate limiting issues. So, you decide to use something like Firehose to write the messages to S3. However, Firehose comes with it's own limits, such as message count & data transfer. A micro-batch can be used to batch up the messages and send them off to firehose when the message count in the batch reaches a certain number, or when a timeout occurs, whichever comes first.

### Dataloader
Fetching data from an external source can be optimized by doing it in batches, especially when establishing the initial connection is expensive. A micro-batch can be used to load data from an external datasource by putting all the requires resources into a single request.