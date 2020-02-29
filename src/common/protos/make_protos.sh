#!/bin/bash

protos_dir="."
gen_dir="gen"
swagger_dir="swagger"
proto_suffix="proto"
proto_gen_suffix="pb.go"
proto_gen_gw_suffix="pb.gw.go"
swagger_suffix="swagger.json"

declare -a proto_names=("echo" "ping_pong" "discovery")

for i in "${proto_names[@]}"
do
    echo "Removing: $protos_dir/$gen_dir/$i.$proto_gen_suffix"
    rm -f $protos_dir/$i.$proto_gen_suffix
    echo "Removing: $protos_dir/$gen_dir/$i.$proto_gen_gw_suffix"
    rm -f $protos_dir/$gen_dir/$i.$proto_gen_gw_suffix
    echo "Removing: $protos_dir/$swagger_dir/$i.$swagger_suffix"
    rm -f $protos_dir/$swagger_dir/$i.$swagger_suffix

    echo "Generating PROTOS: $protos_dir/$i.$proto_suffix"
    protoc -I/usr/local/include -I. \
      -I$GOPATH/src \
      -I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
      --go_out=plugins=grpc:. \
      $protos_dir/$i.$proto_suffix

    mv $protos_dir/$i.$proto_gen_suffix $protos_dir/$gen_dir/

    echo "Generating PROXIES: $protos_dir/$i.$proto_suffix"
    protoc -I/usr/local/include -I. \
        -I$GOPATH/src \
        -I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
        --plugin=protoc-gen-grpc-gateway=$GOPATH/bin/protoc-gen-grpc-gateway \
        --grpc-gateway_out=logtostderr=true:. \
        $protos_dir/$i.$proto_suffix

    mv $protos_dir/$i.$proto_gen_gw_suffix $protos_dir/$gen_dir/

     echo "Generating SWAGGER: $protos_dir/$i.$proto_suffix"
     protoc -I/usr/local/include -I. \
     	-I$GOPATH/src \
     	-I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
     	--plugin=protoc-gen-swagger=$GOPATH/bin/protoc-gen-swagger  \
     	--swagger_out=logtostderr=true:. \
         $protos_dir/$i.$proto_suffix

     mv $protos_dir/$i.$swagger_suffix $protos_dir/$swagger_dir/
done
