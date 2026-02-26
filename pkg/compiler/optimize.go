package compiler

// eliminateDeadFunctions removes FunctionDecl nodes from the AST that are never called.
func eliminateDeadFunctions(stmts []Stmt) []Stmt {
	// 1. Map all function declarations by name
	funcs := make(map[string]*FunctionDecl)
	for _, s := range stmts {
		if f, ok := s.(*FunctionDecl); ok {
			funcs[f.Name] = f
		}
	}

	reachable := make(map[string]bool)
	var worklist []string

	// Helper to mark a function as used and queue it for inspection
	addReachable := func(name string) {
		if !reachable[name] {
			reachable[name] = true
			worklist = append(worklist, name)
		}
	}

	// 2. Identify implicit roots: main, isr
	if _, ok := funcs["main"]; ok {
		addReachable("main")
	}
	if _, ok := funcs["isr"]; ok {
		addReachable("isr")
	}

	// Global initializers might call functions (e.g., int x = init_x();)
	for _, s := range stmts {
		if decl, ok := s.(*VariableDecl); ok && decl.Init != nil {
			calls := make(map[string]bool)
			findCallsExpr(decl.Init, calls)
			for call := range calls {
				addReachable(call)
			}
		}
	}

	// 3. Traverse the worklist to find all transitively reachable functions
	for len(worklist) > 0 {
		curr := worklist[0]
		worklist = worklist[1:]

		fDecl, exists := funcs[curr]
		if !exists {
			// It's likely an intrinsic (e.g., print, enable_interrupts) or undefined; skip it
			continue
		}

		calls := make(map[string]bool)
		findCallsStmt(fDecl.Body, calls)
		for call := range calls {
			addReachable(call)
		}
	}

	// 4. Rebuild the AST, dropping unreachable functions
	var optimized []Stmt
	for _, s := range stmts {
		if f, ok := s.(*FunctionDecl); ok {
			if !reachable[f.Name] {
				continue // Skip dead function
			}
		}
		optimized = append(optimized, s)
	}

	return optimized
}

// findCallsExpr recursively extracts function call names from an expression.
func findCallsExpr(e Expr, calls map[string]bool) {
	if e == nil {
		return
	}
	switch n := e.(type) {
	case *FunctionCall:
		calls[n.Name] = true
		for _, arg := range n.Args {
			findCallsExpr(arg, calls)
		}
	case *BinaryExpr:
		findCallsExpr(n.Left, calls)
		findCallsExpr(n.Right, calls)
	case *LogicalExpr:
		findCallsExpr(n.Left, calls)
		findCallsExpr(n.Right, calls)
	case *UnaryExpr:
		findCallsExpr(n.Right, calls)
	case *PostfixExpr:
		findCallsExpr(n.Left, calls)
	case *IndexExpr:
		findCallsExpr(n.Left, calls)
		for _, idx := range n.Indices {
			findCallsExpr(idx, calls)
		}
	case *MemberExpr:
		findCallsExpr(n.Left, calls)
	case *Literal, *StringLiteral, *VarRef:
		// No function calls here
	}
}

// findCallsStmt recursively extracts function call names from a statement.
func findCallsStmt(s Stmt, calls map[string]bool) {
	if s == nil {
		return
	}
	switch n := s.(type) {
	case *VariableDecl:
		findCallsExpr(n.Init, calls)
	case *Assignment:
		findCallsExpr(n.Left, calls)
		findCallsExpr(n.Value, calls)
	case *ReturnStmt:
		findCallsExpr(n.Expr, calls)
	case *BlockStmt:
		for _, child := range n.Stmts {
			findCallsStmt(child, calls)
		}
	case *IfStmt:
		findCallsExpr(n.Condition, calls)
		findCallsStmt(n.Body, calls)
		findCallsStmt(n.ElseBody, calls)
	case *WhileStmt:
		findCallsExpr(n.Condition, calls)
		findCallsStmt(n.Body, calls)
	case *ForStmt:
		findCallsStmt(n.Init, calls)
		findCallsExpr(n.Cond, calls)
		findCallsStmt(n.Post, calls)
		findCallsStmt(n.Body, calls)
	case *ExprStmt:
		findCallsExpr(n.Expr, calls)
	case *SwitchStmt:
		findCallsExpr(n.Target, calls)
		for _, clause := range n.Cases {
			findCallsExpr(clause.Value, calls)
			for _, child := range clause.Body {
				findCallsStmt(child, calls)
			}
		}
		for _, child := range n.Default {
			findCallsStmt(child, calls)
		}
	case *StructDecl, *AsmStmt, *FunctionDecl:
		// No executable function calls inside these raw declarations/statements
	}
}
