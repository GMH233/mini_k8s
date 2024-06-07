#!/bin/bash

./kubectl delete pod ratings-v1
./kubectl delete pod reviews-v1
./kubectl delete pod reviews-v2
./kubectl delete pod reviews-v3
./kubectl delete pod details-v1
./kubectl delete pod productpage

./kubectl delete service ratings
./kubectl delete service reviews
./kubectl delete service details
./kubectl delete service productpage
