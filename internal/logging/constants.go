package logging

// Standardized field names for structured logging.
// These constants ensure consistency across the application's log output,
// making logs easier to parse, filter, and analyze.
const (
	FieldFile          = "file_path"
	FieldParser        = "parser"
	FieldTransactionID = "transaction_id"
	FieldCategory      = "category"
	FieldReason        = "reason"
	FieldOperation     = "operation"
	FieldStatus        = "status"
	FieldError         = "error"
	FieldDuration      = "duration_ms"
	FieldCount         = "count"
	FieldDelimiter     = "delimiter"
	FieldInputFile     = "input_file"
	FieldOutputFile    = "output_file"
)
