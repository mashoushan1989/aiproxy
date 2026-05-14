import SyntaxHighlighter from 'react-syntax-highlighter/dist/esm/prism-light'
import bash from 'react-syntax-highlighter/dist/esm/languages/prism/bash'
import json from 'react-syntax-highlighter/dist/esm/languages/prism/json'
import { atomDark } from 'react-syntax-highlighter/dist/esm/styles/prism'

SyntaxHighlighter.registerLanguage('bash', bash)
SyntaxHighlighter.registerLanguage('json', json)

const CodeBlock = ({ code, language = 'bash' }: { code: string; language?: string }) => {
    const customizedStyle = {
        ...atomDark,
        'pre[class*="language-"]': {
            ...atomDark['pre[class*="language-"]'],
            backgroundColor: 'transparent',
            margin: 0,
            padding: 0
        }
    }

    return (
        <div
            className="overflow-x-auto"
            style={{
                msOverflowStyle: 'none',
                scrollbarWidth: 'none',
            }}>
            <style dangerouslySetInnerHTML={{
                __html: `
                div::-webkit-scrollbar {
                    width: 0;
                    height: 0;
                }
                div pre::-webkit-scrollbar {
                    width: 0;
                    height: 0;
                }
                div code::-webkit-scrollbar {
                    width: 0;
                    height: 0;
                }
            `}} />
            <SyntaxHighlighter
                language={language}
                style={customizedStyle}
                customStyle={{
                    fontSize: '12px',
                    overflowX: 'auto',
                    msOverflowStyle: 'none',
                    scrollbarWidth: 'none'
                }}
                codeTagProps={{
                    style: {
                        color: 'white'
                    }
                }}
                wrapLines={false}
                lineProps={{ style: { whiteSpace: 'pre' } }}>
                {code}
            </SyntaxHighlighter>
        </div>
    )
}

export default CodeBlock
