#!/usr/bin/env python3

db = "06 A0 FF 09 01 A1 01 09 03 15 00 26 FF 00 75 08 95 40 81 08 09 04 15 00 26 FF 00 75 08 95 40 91 08 C0"
dbe = db.split(" ")

print("var ledgerNanoXReport := []byte{")

for i in dbe:
    print("\t0x{},".format(i))

print("}")
