#!/bin/sh

if [ "${MYSQL_HOST}" = "" ]
then
   echo "No mysql host in enviroment variable \$MYSQL_HOST provided."
   exit 1
fi

if [ "${MYSQL_USER}" = "" ]
then
   echo "No mysql user in enviroment variable \$MYSQL_USER provided."
   exit 1
fi

if [ "${MYSQL_PASSWORD}" = "" ]
then
   echo "No mysql password in enviroment variable \$MYSQL_PASSWORD provided."
   exit 1
fi

if [ "${MYSQL_PORT}" = "" ]
then
   echo "No mysql port in enviroment variable \$MYSQL_PORT provided."
   exit 1
fi

#mysql -h HOST -P PORT_NUMBER -u USERNAME -p
#mysql -h $MYSQL_HOST -P $MYSQL_PORT -u $MYSQL_USER -p$MYSQL_PASSWORD -e ";"
# echo "bevore if"
# if [[ ! mysql -h $MYSQL_HOST -P $MYSQL_PORT -u $MYSQL_USER -p$MYSQL_PASSWORD -e ";" ]] then
#     echo "couldn't connect"
#     exit 1
# fi


counter=0
isconnected=0
while [ "${counter}" -lt 20 ]
do

    DBSTATUS=$(mariadb -h "${MYSQL_HOST}" -P "${MYSQL_PORT}" -u"${MYSQL_USER}" -p"${MYSQL_PASSWORD}" -e "SHOW DATABASES;" 2>&1 )
    if [ "$?" -eq "0" ]
    then
        #echo "Connection ok"
        isconnected=1
        break
    fi

    # if [ ! -n "$DBCONNECTION" ]
    # then
    #     echo "Connection error"
    # fi

    sleep 5

    counter=$(expr "${counter}" + 1)
done

# connection success
if [ ${isconnected} -eq 1 ]
then
    exit 0
fi

# connection failure
if [ ${isconnected} -eq 0 ]
then
    echo "${DBSTATUS}" | tail -n 1
    exit 1
fi

echo "unknown"
exit 0
