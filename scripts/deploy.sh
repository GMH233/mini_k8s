#!/bin/bash

./kubectl apply -f ./test/kubectl/microservice/details.yaml
./kubectl apply -f ./test/kubectl/microservice/productpage.yaml
./kubectl apply -f ./test/kubectl/microservice/ratings.yaml
./kubectl apply -f ./test/kubectl/microservice/reviews-v1.yaml
./kubectl apply -f ./test/kubectl/microservice/reviews-v2.yaml
./kubectl apply -f ./test/kubectl/microservice/reviews-v3.yaml

./kubectl apply -f ./test/kubectl/microservice/details-svc.yaml
./kubectl apply -f ./test/kubectl/microservice/productpage-svc.yaml
./kubectl apply -f ./test/kubectl/microservice/ratings-svc.yaml
./kubectl apply -f ./test/kubectl/microservice/reviews-svc.yaml
