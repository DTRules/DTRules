{
   $0 = toupper($0)
   $1 = gensub("([a-zA-Z])","\\1 ","g",$1)
   print $2 "\t:\t" $1 "\t;"
   
}