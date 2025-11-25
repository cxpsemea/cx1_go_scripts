if(param.Length == 1){
	string relevantKey = param[0] as string;

	CxList integerLiterals = Find_IntegerLiterals();
	CxList methods = Find_Methods();
	CxList objectCreations = Find_ObjectCreations();
	CxList unkRefs = Find_UnknownReference();
	CxList scope = All.NewCxList(integerLiterals, unkRefs, Find_BinaryExpr());

	CxList sizeScope = All.NewCxList();	
	
	if(relevantKey.Equals("Secure")){
		sizeScope.Add(scope.FindByAbstractValue(
			absVal => absVal is IntegerIntervalAbstractValue && 
			(absVal.IncludedIn(new IntegerIntervalAbstractValue(24)) ||
			absVal.IncludedIn(new IntegerIntervalAbstractValue(32)))));
	} else if (relevantKey.Equals("Insecure")){
		sizeScope.Add(scope.FindByAbstractValue(
			absVal => absVal is IntegerIntervalAbstractValue && 
			absVal.IncludedIn(new IntegerIntervalAbstractValue(16))));		
	} else {
		return All.NewCxList();
	}
	
	CxList relevantKeySizes = sizeScope.GetAncOfType<Declarator>();
	relevantKeySizes.Add(unkRefs.FindAllReferences(relevantKeySizes));

	CxList keyParameter = objectCreations.FindByShortName("KeyParameter").FindByParameters(relevantKeySizes);

	CxList cipherInit = methods.FindByMemberAccesses(
		new [] {"PaddedBufferedBlockCipher", "CbcBlockCipher"}, 
		new [] {"Init"}) * keyParameter.GetAncOfType<MethodInvokeExpr>();

	return unkRefs.FindAllReferences(cipherInit.GetTargetOfMembers()).GetMembersOfTarget()
		.FindByType<MethodInvokeExpr>().FindByShortName("DoFinal");
}