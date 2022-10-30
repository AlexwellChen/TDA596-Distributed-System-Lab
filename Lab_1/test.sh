for i in `seq 1 6`; do
curl -d 'entry=t'${i} -X 'POST' 'http://10.1.0.'${i}'/board' &
done
