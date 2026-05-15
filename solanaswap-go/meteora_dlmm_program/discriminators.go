package meteoradlmmprogram

// Instruction discriminators
var (
	Instruction_Swap = [8]byte{
		248,
		198,
		158,
		145,
		225,
		117,
		135,
		200,
	}
	Instruction_Swap2 = [8]byte{
		65,
		75,
		63,
		76,
		235,
		91,
		91,
		136,
	}
	Instruction_SwapExactOut = [8]byte{
		250,
		73,
		101,
		33,
		38,
		207,
		75,
		184,
	}
	Instruction_SwapExactOut2 = [8]byte{
		43,
		215,
		247,
		132,
		137,
		60,
		243,
		81,
	}
	Instruction_SwapWithPriceImpact = [8]byte{
		56,
		173,
		230,
		208,
		173,
		228,
		156,
		205,
	}
	Instruction_SwapWithPriceImpact2 = [8]byte{
		74,
		98,
		192,
		214,
		177,
		51,
		75,
		51,
	}
)
