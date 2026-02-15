-- iCompta Database Schema
-- Reference for camt-csv output format targeting
-- Source: User's iCompta SQLite database
--
-- Key tables for transaction import:
--   ICAccount        - Bank accounts (checking, credit card, savings, loan)
--   ICTransaction    - Individual transactions
--   ICTransactionSplit - Split transactions (category assignment, linked transfers)
--   ICCategory       - Expense/income categories (hierarchical)
--   ICCurrency       - Currency definitions with exchange rates
--   ICBankStatement  - Statement periods
--   ICInvestmentTransactionInfo - Investment transaction metadata
--   ICSecurity       - Securities (stocks, funds)
--   ICQuote          - Security price quotes

-- ICAccount definition

CREATE TABLE ICAccount (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "class" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "icon" BLOB, "iconScale" INTEGER,
    "hidden" INTEGER NOT NULL,
    "parent" TEXT NOT NULL,
    "index" INTEGER NOT NULL,
    "comment" TEXT,
    "currency" TEXT NOT NULL,
    "warningBalance" TEXT NOT NULL,
    "number" TEXT,
    "webSite" TEXT,
    "includedInTotal" INTEGER,
    "type" TEXT,
    "checkingAccountInfo" TEXT,
    "creditCardAccountInfo" TEXT,
    "savingsAccountInfo" TEXT,
    "loanAccountInfo" TEXT,
    "owner" TEXT
);


-- ICBankStatement definition

CREATE TABLE ICBankStatement (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "account" TEXT NOT NULL,
    "date" TEXT NOT NULL,
    "name" TEXT,
    "startingBalance" TEXT NOT NULL,
    "endingBalance" TEXT NOT NULL
);

CREATE INDEX ICBankStatementAccountIndex ON ICBankStatement ("account");


-- ICBudget definition

CREATE TABLE ICBudget (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "class" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "icon" BLOB, "iconScale" INTEGER,
    "hidden" INTEGER NOT NULL,
    "parent" TEXT NOT NULL,
    "index" INTEGER NOT NULL,
    "comment" TEXT,
    "currency" TEXT NOT NULL,
    "accounts" TEXT,
    "remainingTransactionsAccounts" TEXT
);


-- ICBudgetItem definition

CREATE TABLE ICBudgetItem (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "budget" TEXT NOT NULL,
    "index" INTEGER NOT NULL,
    "category" TEXT,
    "amount" TEXT NOT NULL,
    "income" INTEGER NOT NULL,
    "startDate" TEXT NOT NULL,
    "endDate" TEXT,
    "duration" INTEGER NOT NULL, "durationUnit" INTEGER NOT NULL
);

CREATE INDEX ICBudgetItemBudgetIndex ON ICBudgetItem ("budget");


-- ICBudgetItemPeriod definition

CREATE TABLE ICBudgetItemPeriod (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "item" TEXT NOT NULL,
    "index" INTEGER NOT NULL,
    "amount" TEXT NOT NULL,
    "startDate" TEXT NOT NULL,
    "endDate" TEXT NOT NULL
);

CREATE INDEX ICBudgetItemPeriodItemIndex ON ICBudgetItemPeriod ("item");


-- ICBudgetItemPeriodAllocation definition

CREATE TABLE ICBudgetItemPeriodAllocation (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "sender" TEXT NOT NULL,
    "receiver" TEXT NOT NULL,
    "amount" TEXT NOT NULL
);


-- ICCard definition

CREATE TABLE ICCard (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "checkingAccountInfo" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "number" TEXT
);


-- ICCategory definition

CREATE TABLE ICCategory (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "parent" TEXT NOT NULL,
    "index" INTEGER NOT NULL,
    "name" TEXT NOT NULL,
    "icon" BLOB, "iconScale" INTEGER,
    "color" TEXT NOT NULL,
    "expense" INTEGER NOT NULL,
    "income" INTEGER NOT NULL
);


-- ICCheckbook definition

CREATE TABLE ICCheckbook (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "checkingAccountInfo" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "nextCheckNumber" TEXT
);


-- ICCheckingAccountInfo definition

CREATE TABLE ICCheckingAccountInfo (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "account" TEXT NOT NULL
);


-- ICClient definition

CREATE TABLE ICClient (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "class" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "icon" BLOB, "iconScale" INTEGER,
    "hidden" INTEGER NOT NULL,
    "parent" TEXT NOT NULL,
    "currency" TEXT NOT NULL,
    "address" TEXT,
    "phone" TEXT,
    "mail" TEXT
);


-- ICCompany definition

CREATE TABLE ICCompany (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "index" INTEGER NOT NULL,
    "name" TEXT NOT NULL,
    "logo" BLOB, "logoScale" INTEGER,
    "identificationNumber" TEXT,
    "taxesNumber" TEXT,
    "address" TEXT,
    "phone" TEXT,
    "mail" TEXT,
    "webSite" TEXT,
    "nextInvoiceNumber" TEXT,
    "currency" TEXT NOT NULL
);


-- ICCreditCardAccountInfo definition

CREATE TABLE ICCreditCardAccountInfo (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "account" TEXT NOT NULL,
    "number" TEXT,
    "transfer" TEXT,
    "linkedTransaction" TEXT
);


-- ICCurrency definition

CREATE TABLE ICCurrency (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "index" INTEGER NOT NULL,
    "code" TEXT NOT NULL,
    "symbol" TEXT NOT NULL,
    "fractionDigits" INTEGER NOT NULL,
    "changeRate" TEXT NOT NULL
);


-- ICExportDataHandler definition

CREATE TABLE ICExportDataHandler (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "class" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "pluginClass" TEXT NOT NULL,
    "plugin" TEXT,
    "URL_dataURL" TEXT
);


-- ICExportPlugin definition

CREATE TABLE ICExportPlugin (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "class" TEXT NOT NULL,
    "list" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "encoding" TEXT NOT NULL,
    "dateFormat" TEXT NOT NULL,
    "decimalSeparator" TEXT NOT NULL,
    "groupingSeparator" TEXT NOT NULL,
    "transactionsFields" TEXT,
    "transactionsMapping" TEXT,
    "splitsFields" TEXT,
    "splitsMapping" TEXT,
    "mappedValues" TEXT,
    "showSettings" INTEGER NOT NULL,
    "showMapping" INTEGER NOT NULL,
    "CSV_separator" TEXT,
    "CSV_accountNameColumn" TEXT,
    "CSV_accountNumberColumn" TEXT,
    "XML_accountsPath" TEXT,
    "XML_accountNamePath" TEXT,
    "XML_accountNumberPath" TEXT,
    "XML_transactionsPath" TEXT,
    "XML_splitsPath" TEXT,
    "JSON_accountsPath" TEXT,
    "JSON_accountNamePath" TEXT,
    "JSON_accountNumberPath" TEXT,
    "JSON_transactionsPath" TEXT,
    "JSON_splitsPath" TEXT
);


-- ICImportDataProvider definition

CREATE TABLE ICImportDataProvider (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "class" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "pluginClass" TEXT NOT NULL,
    "plugin" TEXT,
    "lastImportDate" TEXT,
    "URL_dataURL" TEXT,
    "OFX_serverURL" TEXT,
    "OFX_login" TEXT,
    "OFX_organization" TEXT,
    "OFX_institutionID" TEXT,
    "OFX_accountType" TEXT,
    "OFX_bankID" TEXT,
    "OFX_brokerID" TEXT,
    "OFX_accountID" TEXT,
    "Linxo_mail" TEXT,
    "Linxo_accountInfos" TEXT
);


-- ICImportPlugin definition

CREATE TABLE ICImportPlugin (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "class" TEXT NOT NULL,
    "list" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "encoding" TEXT NOT NULL,
    "dateFormat" TEXT NOT NULL,
    "decimalSeparator" TEXT,
    "groupingSeparator" TEXT,
    "accountsMapping" TEXT,
    "transactionsMapping" TEXT,
    "splitsMapping" TEXT,
    "mappedValues" TEXT,
    "categorize" INTEGER NOT NULL,
    "setClearedStatusByDefault" INTEGER NOT NULL,
    "applyRules" INTEGER NOT NULL,
    "reconcile" INTEGER NOT NULL,
    "reconcileUsingName" INTEGER NOT NULL,
    "reconcileUsingDate" INTEGER NOT NULL,
    "numberOfDays" INTEGER NOT NULL,
    "updateDate" INTEGER NOT NULL,
    "updateValueDate" INTEGER NOT NULL,
    "updateName" INTEGER NOT NULL,
    "updateComment" INTEGER NOT NULL,
    "updateStatus" INTEGER NOT NULL,
    "showSettings" INTEGER NOT NULL,
    "showMapping" INTEGER NOT NULL,
    "CSV_separator" TEXT,
    "CSV_firstLine" INTEGER,
    "CSV_hasHeader" INTEGER,
    "CSV_accountNameColumn" TEXT,
    "CSV_accountNumberColumn" TEXT,
    "CSV_splitsCountColumn" TEXT,
    "XML_accountsPath" TEXT,
    "XML_accountNamePath" TEXT,
    "XML_accountNumberPath" TEXT,
    "XML_transactionsPath" TEXT,
    "XML_splitsPath" TEXT,
    "JSON_accountsPath" TEXT,
    "JSON_accountNamePath" TEXT,
    "JSON_accountNumberPath" TEXT,
    "JSON_transactionsPath" TEXT,
    "JSON_splitsPath" TEXT
);


-- ICImportPluginsList definition

CREATE TABLE ICImportPluginsList (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "pluginsClass" TEXT NOT NULL
);


-- ICInterestRate definition

CREATE TABLE ICInterestRate (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "savingsAccountInfo" TEXT NOT NULL,
    "date" TEXT NOT NULL,
    "rate" TEXT NOT NULL
);


-- ICInvestmentTransactionInfo definition

CREATE TABLE ICInvestmentTransactionInfo (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "type" TEXT,
    "security" TEXT,
    "shares" TEXT,
    "commission" TEXT
);


-- ICInvoice definition

CREATE TABLE ICInvoice (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "company" TEXT NOT NULL,
    "type" TEXT NOT NULL,
    "number" TEXT NOT NULL,
    "date" TEXT NOT NULL,
    "dueDate" TEXT,
    "title" TEXT NOT NULL,
    "comment" TEXT,
    "client" TEXT,
    "currency" TEXT NOT NULL,
    "discount" TEXT
);


-- ICInvoiceItem definition

CREATE TABLE ICInvoiceItem (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "invoice" TEXT NOT NULL,
    "index" INTEGER NOT NULL,
    "title" TEXT NOT NULL,
    "details" TEXT,
    "amount" TEXT,
    "amountIncludesTaxes" INTEGER NOT NULL,
    "quantity" TEXT,
    "taxesName" TEXT,
    "taxesRate" TEXT,
    "taxesCategory" TEXT,
    "free" INTEGER NOT NULL,
    "category" TEXT
);

CREATE INDEX ICInvoiceItemInvoiceIndex ON ICInvoiceItem ("invoice");


-- ICInvoiceTemplate definition

CREATE TABLE ICInvoiceTemplate (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "index" INTEGER NOT NULL,
    "name" TEXT NOT NULL,
    "content" TEXT
);


-- ICLibrary definition

CREATE TABLE ICLibrary (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT
);


-- ICLoanAccountInfo definition

CREATE TABLE ICLoanAccountInfo (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "account" TEXT NOT NULL,
    "rate" TEXT NOT NULL,
    "transfer" TEXT,
    "linkedTransaction" TEXT,
    "linkedTransactionAmount" TEXT
);


-- ICOwner definition

CREATE TABLE ICOwner (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "parent" TEXT,
    "index" INTEGER NOT NULL,
    "person" TEXT NOT NULL,
    "ratio" TEXT NOT NULL
);

CREATE INDEX ICOwnerParentIndex ON ICOwner ("parent");


-- ICQuote definition

CREATE TABLE ICQuote (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "security" TEXT NOT NULL,
    "date" TEXT NOT NULL,
    "value" TEXT NOT NULL
);

CREATE INDEX ICQuoteSecurityIndex ON ICQuote ("security");


-- ICReport definition

CREATE TABLE ICReport (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "class" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "icon" BLOB, "iconScale" INTEGER,
    "hidden" INTEGER NOT NULL,
    "parent" TEXT,
    "index" INTEGER NOT NULL,
    "currency" TEXT,
    "transactionsCondition" TEXT,
    "splitsCondition" TEXT,
    "usesValueDate" INTEGER,
    "type" TEXT,
    "groupingKeyPaths" TEXT,
    "startDate" TEXT,
    "period" INTEGER, "periodUnit" INTEGER,
    "addTotalPeriod" INTEGER,
    "addAveragePeriod" INTEGER,
    "groupsByParent" INTEGER,
    "comparisonKeyPath" TEXT
);


-- ICSavingsAccountInfo definition

CREATE TABLE ICSavingsAccountInfo (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "account" TEXT NOT NULL,
    "interest" TEXT
);


-- ICScheduledTransaction definition

CREATE TABLE ICScheduledTransaction (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "account" TEXT NOT NULL,
    "date" TEXT NOT NULL,
    "valueDate" TEXT,
    "index" INTEGER NOT NULL,
    "name" TEXT NOT NULL,
    "comment" TEXT,
    "useSumOfSplits" INTEGER NOT NULL,
    "amount" TEXT,
    "amountWithoutTaxes" TEXT,
    "taxesRate" TEXT,
    "payee" TEXT,
    "type" TEXT,
    "number" TEXT,
    "links" BLOB,
    "highlightColor" TEXT,
    "latitude" REAL,
    "longitude" REAL,
    "investmentTransactionInfo" TEXT,
    "accountInfo" TEXT,
    "frequency" INTEGER, "frequencyUnit" INTEGER,
    "numberOfOccurrences" INTEGER,
    "nextOccurrence" INTEGER NOT NULL
);


-- ICScheduledTransactionSplit definition

CREATE TABLE ICScheduledTransactionSplit (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "transaction" TEXT NOT NULL,
    "index" INTEGER NOT NULL,
    "amount" TEXT,
    "amountWithoutTaxes" TEXT,
    "taxesRate" TEXT,
    "taxesCategory" TEXT,
    "ratio" TEXT,
    "comment" TEXT,
    "project" TEXT,
    "category" TEXT,
    "linkedSplit" TEXT,
    "ignoredInBudgets" INTEGER NOT NULL,
    "invoice" TEXT,
    "ignoredInReports" INTEGER NOT NULL,
    "ignoredInAverageBalance" INTEGER NOT NULL,
    "refund" INTEGER NOT NULL,
    "usesAccountOwners" INTEGER NOT NULL
);

CREATE INDEX ICScheduledTransactionSplitTransactionIndex ON ICScheduledTransactionSplit ("transaction");


-- ICSecurity definition

CREATE TABLE ICSecurity (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "name" TEXT NOT NULL,
    "symbol" TEXT,
    "currency" TEXT NOT NULL,
    "hidden" INTEGER NOT NULL
);


-- ICTax definition

CREATE TABLE ICTax (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "index" INTEGER NOT NULL,
    "name" TEXT NOT NULL,
    "rate" TEXT NOT NULL
);


-- ICTransaction definition

CREATE TABLE ICTransaction (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "account" TEXT NOT NULL,
    "date" TEXT NOT NULL,
    "valueDate" TEXT,
    "index" INTEGER NOT NULL,
    "name" TEXT NOT NULL,
    "comment" TEXT,
    "useSumOfSplits" INTEGER NOT NULL,
    "amount" TEXT,
    "amountWithoutTaxes" TEXT,
    "taxesRate" TEXT,
    "payee" TEXT,
    "type" TEXT,
    "number" TEXT,
    "links" BLOB,
    "highlightColor" TEXT,
    "latitude" REAL,
    "longitude" REAL,
    "investmentTransactionInfo" TEXT,
    "scheduledTransaction" TEXT,
    "occurrence" INTEGER,
    "status" TEXT NOT NULL,
    "budgetItemPeriod" TEXT,
    "statement" TEXT,
    "externalID" TEXT
);

CREATE INDEX ICTransactionAccountIndex ON ICTransaction ("account");
CREATE INDEX ICTransactionScheduledTransactionIndex ON ICTransaction ("scheduledTransaction");
CREATE INDEX ICTransactionBudgetItemPeriodIndex ON ICTransaction ("budgetItemPeriod");


-- ICTransactionSplit definition

CREATE TABLE ICTransactionSplit (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "transaction" TEXT NOT NULL,
    "index" INTEGER NOT NULL,
    "amount" TEXT,
    "amountWithoutTaxes" TEXT,
    "taxesRate" TEXT,
    "taxesCategory" TEXT,
    "ratio" TEXT,
    "comment" TEXT,
    "project" TEXT,
    "category" TEXT,
    "linkedSplit" TEXT,
    "ignoredInBudgets" INTEGER NOT NULL,
    "invoice" TEXT,
    "ignoredInReports" INTEGER NOT NULL,
    "ignoredInAverageBalance" INTEGER NOT NULL,
    "refund" INTEGER NOT NULL,
    "usesAccountOwners" INTEGER NOT NULL,
    "scheduledSplit" TEXT,
    "budgetItemPeriod" TEXT
);

CREATE INDEX ICTransactionSplitTransactionIndex ON ICTransactionSplit ("transaction");
CREATE INDEX ICTransactionSplitScheduledSplitIndex ON ICTransactionSplit ("scheduledSplit");


-- LAAction definition

CREATE TABLE LAAction (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "rule" TEXT NOT NULL,
    "index" INTEGER NOT NULL,
    "name" TEXT NOT NULL,
    "parameter1" TEXT,
    "parameter2" TEXT,
    "parameter3" TEXT,
    "parameter4" TEXT,
    "parameter5" TEXT
);

CREATE INDEX LAActionRuleIndex ON LAAction ("rule");


-- LACondition definition

CREATE TABLE LACondition (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "class" TEXT NOT NULL,
    "parent" TEXT,
    "index" INTEGER NOT NULL,
    "type" INTEGER,
    "keyPath" TEXT,
    "operator" TEXT,
    "parameter" TEXT
);

CREATE INDEX LAConditionParentIndex ON LACondition ("parent");


-- LADeletedID definition

CREATE TABLE LADeletedID (
    "ID" TEXT UNIQUE NOT NULL,
    "date" TEXT
);


-- LAFiltersBarData definition

CREATE TABLE LAFiltersBarData (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "name" TEXT NOT NULL,
    "values" TEXT,
    "condition" TEXT
);


-- LAMetadata definition

CREATE TABLE LAMetadata (
    "version" TEXT NOT NULL,
    "password" TEXT,
    "localParameters" TEXT
);


-- LAParameter definition

CREATE TABLE LAParameter (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "class" TEXT NOT NULL,
    "string" TEXT,
    "dateType" TEXT,
    "date" TEXT,
    "dateOperator" TEXT,
    "dateDuration" INTEGER, "dateDurationUnit" INTEGER,
    "number" TEXT,
    "color" TEXT,
    "constant" TEXT,
    "object" TEXT
);


-- LARule definition

CREATE TABLE LARule (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "class" TEXT NOT NULL,
    "parent" TEXT NOT NULL,
    "index" INTEGER NOT NULL,
    "name" TEXT NOT NULL,
    "active" INTEGER NOT NULL,
    "condition" TEXT
);


-- LASavedCondition definition

CREATE TABLE LASavedCondition (
    "sqlID" INTEGER PRIMARY KEY,
    "ID" TEXT UNIQUE NOT NULL,
    "lastModificationDate" TEXT,
    "filtersBarData" TEXT NOT NULL,
    "index" INTEGER NOT NULL,
    "name" TEXT NOT NULL,
    "condition" TEXT NOT NULL
);


-- LASynchronizationHandler definition

CREATE TABLE LASynchronizationHandler (
    "description" TEXT NOT NULL,
    "identifier" TEXT,
    "active" INTEGER NOT NULL,
    "lastSynchronizationDate" TEXT
);
