package chains

import (
	"errors"
	"log"

	"github.com/asolpshinning/chaingpt/agents/codeAgents"
	"github.com/asolpshinning/chaingpt/entity"
	entityTools "github.com/asolpshinning/chaingpt/tools/db-tools"
)

var basePrompt = `
	You are a powerful AI that can convert user's text or chat to SQL code, and the code you provide can be used to query a database.
	You will be provided back the result of the query as an observation. You will then interpret the result and 
	provide a response back to the user. Below will show how you are processing your thoughts clearly
	and thinking out loud. It is going to be in the following format. The final answer you need to give will be an observation. Only reply with 
	ONE final observation. DO NOT reply with any actions. The actions are only for you to see how you are processing your thoughts. 
		Observation: <observation>
		Action: <action>
		Observation: <observation>
		Action: <action>
		Observation: ??

`

// This is Chain-type function that has the purpose of helping achieve natural language conversation with a database.
// The tool set provided must include a databaseQuery tool (index = 0) and a gpt tool (index = 1).
func ChatWithDatabase(from *entity.Agent, input *entity.AgentResponse, tools []*entity.Tool) (*entity.ChainResponse, error) {
	// make sure the tools is not empty or 1 in length). If it is then return an error message
	if len(tools) != 2 {
		err := errors.New("two tools need to be provided to the chain")
		log.Println(err)
		return nil, err
	}

	// if tools[0].Type is not "gpt", then return an error message
	if tools[0].Type != "gpt" {
		err := errors.New("the first tool in your tool set is not a gpt tool")
		log.Println(err)
		return nil, err
	}

	// if tools[0].Value is not "chatGPT", then return an error message
	if tools[0].Value != "chatGPT" && tools[0].Value != "flanT5" {
		err := errors.New("the gpt tool type provided to the chain is not yet supported")
		log.Println(err)
		return nil, err
	}

	// check if tools[1].Type is "databaseQuery", then let agents.textToSQL do the work
	if tools[1].Type != "databaseQuery" {
		err := errors.New("the second tool in your tool set is not a database query tool")
		log.Println(err)
		return nil, err
	}
	if tools[1].Value != "postgres" {
		err := errors.New("the databaseQuery tool type provided to the chain is not yet supported")
		log.Println(err)
		return nil, err
	}

	// send the input.Output as the text to agents.textToSQL and the tool is tools[0]
	basePrompt += "Observation: " + "User has provided this text =>  " + input.Output + "\n"
	basePrompt += "Action: " + "Converting the text user has provided to SQL code. \n"
	resFromTextToSQL, err := codeAgents.TextToSQL(input.Output, tools[0])
	if err != nil {
		log.Println(err)
		return nil, err
	}

	if !resFromTextToSQL.Satisfactory {
		// send the resFromTextToSQL.Output to the user
		response := &entity.ChainResponse{
			Response: resFromTextToSQL.Output,
		}
		return response, nil
	}
	basePrompt += "Observation: " + "`" + resFromTextToSQL.Output + "`" + "\n"
	basePrompt += "Action: " + "Running the SQL code to the database to get the result. \n"

	// send the resulting SQL code to the agent that will run it against a database
	resultFromDB, err := entityTools.RunPostgresQuery(resFromTextToSQL.Output)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	basePrompt += "Observation: " + "Result from the database => " + resultFromDB + "\n"

	basePrompt += "Action: Converting the result observed above to good response in english back to user. " + "\n"

	basePrompt += "Now answer the user's question. They do not have answer yet: " + input.Output + "\n"

	englishAgentResponse, err := codeAgents.QueryResultToEnglish(basePrompt, tools[0])
	if err != nil {
		log.Println(err)
		return nil, err
	}

	chainResponse := &entity.ChainResponse{
		Response: englishAgentResponse.Output,
	}

	//log.Println(basePrompt)
	log.Println("THIS IS THE END OF THE CHAIN")

	return chainResponse, nil
}
