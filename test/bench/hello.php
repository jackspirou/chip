<?php

$start = microtime(TRUE);

echo "Hello world!.";

$end = microtime(TRUE);

$duration = ($end - $start) / 1000;

echo $duration;
