package compiler

import "testing"

// simpleSource is a minimal C program used for benchmarking the fast path.
const simpleSource = `
int add(int a, int b) {
	return a + b;
}

int main() {
	int x = add(3, 4);
	return x;
}
`

// complexSource is a larger program exercising structs, arrays, loops,
// pointer arithmetic, and recursive function calls.
const complexSource = `
struct Point {
	int x;
	int y;
};

int abs_val(int n) {
	if (n < 0) {
		return -n;
	}
	return n;
}

int sum_array(int* arr, int len) {
	int total = 0;
	int i = 0;
	while (i < len) {
		total = total + arr[i];
		i = i + 1;
	}
	return total;
}

int dot_product(int* a, int* b, int len) {
	int result = 0;
	int i = 0;
	for (i = 0; i < len; i = i + 1) {
		result = result + (a[i] * b[i]);
	}
	return result;
}

int fib(int n) {
	if (n == 0) { return 0; }
	if (n == 1) { return 1; }
	return fib(n - 1) + fib(n - 2);
}

int max_in_array(int* arr, int len) {
	int best = arr[0];
	int i = 1;
	while (i < len) {
		if (arr[i] > best) {
			best = arr[i];
		}
		i = i + 1;
	}
	return best;
}

int main() {
	int arr[8];
	arr[0] = 3;
	arr[1] = 1;
	arr[2] = 4;
	arr[3] = 1;
	arr[4] = 5;
	arr[5] = 9;
	arr[6] = 2;
	arr[7] = 6;

	int s = sum_array(arr, 8);
	int m = max_in_array(arr, 8);
	int f = fib(8);
	int a = abs_val(-42);

	int vec_a[4];
	int vec_b[4];
	vec_a[0] = 1; vec_a[1] = 2; vec_a[2] = 3; vec_a[3] = 4;
	vec_b[0] = 4; vec_b[1] = 3; vec_b[2] = 2; vec_b[3] = 1;
	int dp = dot_product(vec_a, vec_b, 4);

	return s + m + f + a + dp;
}
`

// --- Lex benchmarks ---

func BenchmarkLex_Simple(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Lex(simpleSource)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLex_Complex(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Lex(complexSource)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// --- Parse benchmarks ---
// Tokens are pre-computed outside the timed region.

func BenchmarkParse_Simple(b *testing.B) {
	tokens, err := Lex(simpleSource)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Parse(tokens, simpleSource)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParse_Complex(b *testing.B) {
	tokens, err := Lex(complexSource)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Parse(tokens, complexSource)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// --- Generate (code generation) benchmarks ---
// Tokens and AST are pre-computed outside the timed region.

func BenchmarkGenerate_Simple(b *testing.B) {
	tokens, err := Lex(simpleSource)
	if err != nil {
		b.Fatal(err)
	}
	stmts, err := Parse(tokens, simpleSource)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Generate(stmts, NewSymbolTable())
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGenerate_Complex(b *testing.B) {
	tokens, err := Lex(complexSource)
	if err != nil {
		b.Fatal(err)
	}
	stmts, err := Parse(tokens, complexSource)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Generate(stmts, NewSymbolTable())
		if err != nil {
			b.Fatal(err)
		}
	}
}

// --- Full pipeline benchmarks (Lex + Parse + Generate) ---

func BenchmarkCompilerPipeline_Simple(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		tokens, err := Lex(simpleSource)
		if err != nil {
			b.Fatal(err)
		}
		stmts, err := Parse(tokens, simpleSource)
		if err != nil {
			b.Fatal(err)
		}
		_, err = Generate(stmts, NewSymbolTable())
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompilerPipeline_Complex(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		tokens, err := Lex(complexSource)
		if err != nil {
			b.Fatal(err)
		}
		stmts, err := Parse(tokens, complexSource)
		if err != nil {
			b.Fatal(err)
		}
		_, err = Generate(stmts, NewSymbolTable())
		if err != nil {
			b.Fatal(err)
		}
	}
}
