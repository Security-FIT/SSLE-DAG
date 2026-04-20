// Author: Tomáš Hladký
// Master thesis: Design an Experimental PoS DAG-based Blockchain Consensual Protocol
// Brno University of Technology, Faculty of Information Technology, 2025

package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	MAIN          = "MAIN"
	CFG_LOG       = "CFG"
	BTSTRP_LOG    = "BTSTRP"
	P2P_SETUP_LOG = "P2P_SETUP"
	P2P_LOG       = "P2P"
	ZKP_LOG       = "ZKP"
	RMQ_LOG       = "RMQ"
	BCHAIN_LOG    = "BCHAIN"
	DAG_LOG       = "DAG"
)

func logTimeContext(context string) {
	currentTime := time.Now().Format("15:04:05.000")

	var builderFixPart strings.Builder
	builderFixPart.WriteString("[%s] - \u001B[32m[%s]\u001B[0m: ")
	fmt.Printf(builderFixPart.String(), currentTime, context)
}

func logColor(format string, color string, v ...any) {
	var builderDynamicPart strings.Builder
	builderDynamicPart.WriteString(color)
	builderDynamicPart.WriteString(format)
	builderDynamicPart.WriteString("\u001B[0m\n")

	fmt.Printf(builderDynamicPart.String(), v...)
}

func logInfo(context string, format string, v ...any) {
	logTimeContext(context)
	logColor(format, "\u001B[94m", v...)
}

func logInfoBold(context string, format string, v ...any) {
	logTimeContext(context)
	logColor(format, "\u001B[93m", v...)
}

func logError(context string, format string, v ...any) {
	logTimeContext(context)
	logColor(format, "\u001B[31m", v...)
}

func logDebug(context string, format string, v ...any) {
	if disableDebug {
		return
	}
	logTimeContext(context)
	logColor(format, "\u001B[93m", v...)
}

func logFatal(context string, format string, v ...any) {
	logError(context, format, v...)
	os.Exit(1)
}
