SSA Design
----------

All `chip` code is parsed into static single assignment form (SSA). The SSA
intermediate representation (IR) is a form of three address code (TAC).

At first the goal was to use some standardized IR format such as what LLVM
provides, or ARM. That idea quickly melted away since no native go packages
currently support those IR formats (other than the code for golang itself).
Next on the list was MIPS, but upon inspection MIPS still had extra complexity.
This has lead me to use a stripped down perverted subset of MIPS.

In the future the goal will be to move back up the IR chain:
CHIP -> MIPS -> ARM -> LLVM.
