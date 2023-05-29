#!/bin/bash


REG_SIZES="64 128 256 512 1024"
# REG_SIZES="64"



FUNCTIONS=""
FUNCTIONS="${FUNCTIONS} AES-G AES-N AES-P"


# ##### Preprocess results

# go build  -o main main.go
# rm -f res_all.csv

# for fn in $FUNCTIONS
# do
#     for rs in $REG_SIZES
#     do
#         # ./decode_inst_trace.py $RESULTS_DIR/$bm/$fn/$MONITOR.gz &
#         ./main -wl $fn -inv 15   -region-size $rs -o res_tmp.csv
#         cat res_tmp.csv >> res_all.csv
#     done
# done





#######################
REG_SIZES="64"

INVOCATIONS="15 16 17 18"
INVOCATIONS="13 14 19"
INVOCATIONS="20 21 22 23"


INVOCATIONS="13 14 19 20 21 22 23"

# INVOCATIONS="13 14 19 20 21 22 23 24 25 26 27 28 29 "



FUNCTIONS=""
# FUNCTIONS="${FUNCTIONS} AES-G AES-N AES-P"
# FUNCTIONS="${FUNCTIONS} Auth-G Auth-N Auth-P"
# FUNCTIONS="${FUNCTIONS} Fib-G Fib-N Fib-P"

# FUNCTIONS="${FUNCTIONS} Email-P RecO-P Curr-N Pay-N Ship-G"
FUNCTIONS="${FUNCTIONS} Geo-G Prof-G Rate-G RecH-G Res-G User-G"


CONFIG=inst_PF
CONFIG=inst_noPF
CONFIG=data_PF
# CONFIG=data_noPF

CONFIGS=""

# CONFIGS="${CONFIGS} data_PF"
# CONFIGS="${CONFIGS} data_noPF"

# CONFIGS="${CONFIGS} inst_PF"
# CONFIGS="${CONFIGS} inst_noPF"
CONFIGS="${CONFIGS} inst_PF_bpBTB"
CONFIGS="${CONFIGS} inst_PF_noBPU"

# CONFIGS="${CONFIGS} inst_noPF"



CONFIGS="uncond"


##### Preprocess results

# go build  -o main replay_similarity.go

# for cfg in $CONFIGS
# do
#   rm -f res_$cfg.csv


#   for fn in $FUNCTIONS
#   do
#     for rs in $REG_SIZES
#     do
#       for iv in $INVOCATIONS
#       do
#         # ./decode_inst_trace.py $RESULTS_DIR/$bm/$fn/$MONITOR.gz &
#         ./main -wl $fn -inv $iv  -file data_$cfg.json -region-size $rs -o res_tmp.csv;
#         cat res_tmp.csv >> res_$cfg.csv
#       done
#     done
#   done


# done

# join

cfg="uncond_"
cfg="taken_"



rm -f res_$cfg.csv


go build  -o main edit_distance.go

for fn in $FUNCTIONS
do
    # for iv in $INVOCATIONS
    for iv in {13..33}
    do
      # ./decode_inst_trace.py $RESULTS_DIR/$bm/$fn/$MONITOR.gz &
      (
        tmpf=$(mktemp);
        ./main -wl $fn -inv $iv  -file data_$cfg.json -o $tmpf;
        cat $tmpf >> res_$cfg.csv;
        rm $tmpf;
      ) &
    done
done


wait
