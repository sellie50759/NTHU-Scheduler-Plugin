.PHONY: build deploy

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildvcs=false -o=bin/my-scheduler ./cmd/scheduler

buildLocal:
	docker build . -t my-scheduler:local

loadImage:
	kind load docker-image my-scheduler:local

deploy:
	helm install scheduler-plugins charts/ 

remove:
	helm uninstall scheduler-plugins

testPrefilter:
	kubectl create -f test/prefilter.yaml
	@sleep 2
	kubectl get po -o wide
	kubectl get pods --no-headers -o custom-columns=":metadata.name" | grep my-scheduler \
	 | xargs kubectl logs | grep log
	# $(MAKE) testClean

testLeastMode:
	kubectl create -f test/least_mode.yaml
	@sleep 2
	kubectl get po -o wide
	kubectl get pods --no-headers -o custom-columns=":metadata.name" | grep my-scheduler \
	 | xargs kubectl logs | grep log
	# $(MAKE) testClean

testMostMode:
	kubectl create -f test/most_mode.yaml
	@sleep 2
	kubectl get po -o wide
	kubectl get pods --no-headers -o custom-columns=":metadata.name" | grep my-scheduler \
	 | xargs kubectl logs | grep log
	# $(MAKE) testClean

testClean:
	kubectl delete pod --all

clean:
	rm -rf bin/
