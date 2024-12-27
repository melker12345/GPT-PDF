# PDF Reader with GPT Integration

This application allows users to read PDF files and analyze their content using GPT. Users can navigate through PDF pages and get detailed explanations of the content using OpenAI's GPT API.

## Features

- PDF file upload and viewing
- Page navigation
- GPT-powered content analysis
- Split view showing PDF content and GPT analysis side by side

## Prerequisites

- Go 1.21 or later
- OpenAI API key

## Setup

1. Set your OpenAI API key as an environment variable:
```bash
export OPENAI_API_KEY='your-api-key-here'
```

2. Install dependencies:
```bash
go mod tidy
```

3. Run the application:
```bash
go run main.go
```

## Usage

1. Click "Upload PDF" to select and load a PDF file
2. Use "Previous Page" and "Next Page" buttons to navigate through the PDF
3. Click "Analyze with GPT" to get a detailed explanation of the current page's content
4. The GPT analysis will appear in the right panel

## Note

Make sure you have a valid OpenAI API key set in your environment variables before using the GPT analysis feature.




We don't need to extract the text from the PDF and send it to GPT. We can just send the image to GPT and get the response.

we should render the pdf using a pdf reader library or what ever it is called. this meand that we simply render the pdf using a pdf reader library and then send the image to GPT and get the response.