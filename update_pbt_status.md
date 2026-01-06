# PBT Status Update

## Property 8: Consistent CSV output format - FAILED

**Test**: TestProperty_ConsistentCSVOutputFormat
**Status**: FAILED
**Iterations**: 100/100 failed

**Issue**: The PDF parser CSV output format doesn't match the expected headers from the test.

**Current PDF parser CSV headers:**
```
BookkeepingNumber,Status,Date,ValueDate,Name,PartyName,PartyIBAN,Description,RemittanceInfo,Amount,CreditDebit,IsDebit,Debit,Credit,Currency,AmountExclTax,AmountTax,TaxRate,Recipient,InvestmentType,Number,Category,Type,Fund,NumberOfShares,Fees,IBAN,EntryReference,Reference,AccountServicer,BankTxCode,OriginalCurrency,OriginalAmount,ExchangeRate
```

**Missing expected headers:**
- DebitAmount
- CreditAmount  
- SubCategory
- Payee
- PayeeIBAN
- Payer
- PayerIBAN
- RecipientIBAN

**Root Cause**: The test expects a different CSV format than what the PDF parser currently produces. The PDF parser uses the common CSV format from `internal/common/csv.go`, but the test expects additional headers that are not included in that format.

**Failing Example**: All 100 iterations failed with the same missing headers issue.

**Next Steps**: Need user direction on whether to:
1. Update the PDF parser to include the missing headers, OR
2. Update the test expectations to match the actual PDF parser CSV format