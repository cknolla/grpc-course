genGreet:
	protoc greet/greetpb/greet.proto --go_out=plugins=grpc:.

genCalc:
	protoc calculator/calculatorpb/calculator.proto --go_out=plugins=grpc:.

genPrime:
	protoc prime/primepb/prime.proto --go_out=plugins=grpc:.