module webtyp.com/ddlc

go 1.25.2

// Leaf guarantee: webtyp.com/ddlc must remain a leaf package in the SQL ecosystem.
// It must depend ONLY on webtyp/model and webtyp/fmt to keep it portable for WASM/frontend.
require (
	webtyp.com/fmt v0.25.7
	webtyp.com/model v0.1.7
)

require webtyp.com/tui v0.1.2
