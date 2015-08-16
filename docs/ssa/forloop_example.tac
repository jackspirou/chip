#
# var count
#
# for i := 0; i < 10; i++ {
#   count += i
# }
#
# print count
#

:main                   # main label

li t0 $z                # initialize count to zero

li $t1 0                # initialize i to zero
:loop1                  # top of for loop
bge  $t1 10 exitloop2   # conditional for loop break
add  $t0 $t0 $t1        # add i to count
addi $t0 $t0 1          # increase i by 1
goto loop1              # jump back to the top of the for loop
:exitloop2              # exit/ break for loop

print t0                # print count result
