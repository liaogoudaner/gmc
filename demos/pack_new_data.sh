#!/bin/bash

set -e

PWD=`pwd`
ADIR="${PWD}/../../mygmcadmin"

cp -f ../app/web_new.toml new_web/conf/app.toml
cp -f ../app/api_new.toml new_api/conf/app.toml

rm -rf new_web/gmcrun.toml
rm -rf new_api/gmcrun.toml
rm -rf new_simple_api/gmcrun.toml
rm -rf ${ADIR}/gmcrun.toml

tar zcf new_web.tar.gz -C new_web/ .
tar zcf new_api.tar.gz -C new_api/ .
tar zcf new_simple_api.tar.gz -C new_simple_api/ .
tar zcf new_admin.tar.gz -C ${ADIR}/ .


webdata=`base64 -i new_web.tar.gz`
apidata=`base64 -i new_api.tar.gz`
simpleapidata=`base64 -i new_simple_api.tar.gz`
admindata=`base64 -i new_admin.tar.gz`

cat <<EOF >../../gmct/module/new/data.go
package newx

var (
	webData = "$webdata"
	apiData = "$apidata"
	simpleapiData="$simpleapidata"
	adminData="$admindata"
)
EOF

rm -rf new_web.tar.gz
rm -rf new_api.tar.gz
rm -rf new_simple_api.tar.gz
rm -rf new_admin.tar.gz