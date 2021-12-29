## DIY2 for availability new joinee- 

1. Ability to use the library as a singleton in a go service
   - Implement as an interface
2. Ability to define CPUs which the service can use
3. Ability to define goroutine pool which the service can use
   - No of maximum goroutines that can be active at one point of time
4. Ability to hold incoming requests in a FIFO queue if no goroutine is available
   - Explore using buffered channels/slices
5. Ability to drop requests after certain period if not served within a timeframe ( should be configurable by the client )
6. Unit tests coverage should be 85%+
   - Explore inbuilt testing library
   - Avoid mocks but if you want to explore, look at go-mock
7. Create a simple docker file and run the unit tests inside that

# go-pool

This GoLang library manages a pool of goroutines which has got the ability to define the number of CPUs that the services can use, the number of go routines that should run in parallel is configurable and the functionality that has to performed can also be defined.

### Usage- 

1. Install the dependancy
   ``` shell
   go get github.com/nitinks-dnz/go-pool
   ```
2. Initialize the pool
   ```go
   pool := go_pool.Initialize(nCPUs, nRoutines, fun(input interface{}) interface{}{
       // ToDo - Define some functionality to be performed
   })
   defer pool.Close()
   ```
3. Adding jobs to the pool
   ```go
   output, error := pool.Process(input)
   ```
4. Adding jobs to the pool with timeout
   ```go
   output, eerror := pool.ProcessWithExpiry(input, time.Duration(time.Second))
   ```

### Testing - 
1. Build the docker image
   ```shell
   docker build --tag go-pool .
   ```
2. Execute all the test cases
   ```shell
   docker run go-pool go test ./...               //Silently
   docker run go-pool go test -v ./...            //With verbose logs
   ```
3. To view the test coverage report
   ```shell
   docker run go-pool go test -cover ./...
   ```