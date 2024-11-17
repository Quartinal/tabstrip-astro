import fs, { readFileSync, writeFileSync } from "fs";
import ls from "ls";
import { resolve } from "path";

const importedHtmlJsFiles = [
    "./tab.html.js",
    "./tab_list.html.js",
    "./tab_group.html.js",
    "./alert_indicators.html.js",
    "./alert_indicator.html.js"
];

function replaceLineInFile(filePath, lineNumber, newLineContent) {
    // Read the file content
    fs.readFile(filePath, 'utf8', (err, data) => {
        if (err) {
            console.error('Error reading the file:', err);
            return;
        }

        // Split content by lines
        const lines = data.split('\n');

        // Ensure the line number is within bounds
        if (lineNumber < 1 || lineNumber > lines.length) {
            console.error('Invalid line number.');
            return;
        }

        // Replace the specific line
        lines[lineNumber - 1] = newLineContent;

        // Join the lines back into a single string
        const updatedContent = lines.join('\n');

        // Write the updated content back to the file
        fs.writeFile(filePath, updatedContent, 'utf8', (err) => {
            if (err) {
                console.error('Error writing to the file:', err);
                return;
            }
        });
    });
}

for (const file of ls("./out/*.ts")) {
    const fullPath = resolve(import.meta.dirname, file.full);
    const readFile = readFileSync(fullPath).toString();
    const replacedFile = readFile.replace("import '/strings.m.js'", "import './strings.m.js'");
    writeFileSync(fullPath, replacedFile);

    for (const importedHtmlJsFile of importedHtmlJsFiles) {
        const newReadFile = readFileSync(fullPath).toString();

        if (newReadFile.includes(`import {getTemplate} from '${importedHtmlJsFile}'`)) {
            const replacedHtmlJsFile = newReadFile.replace(`import {getTemplate} from '${importedHtmlJsFile}'`, `import {getTemplateHtml} from '${importedHtmlJsFile}'`).replace("getTemplate()", "getTemplateHtml()");
            writeFileSync(fullPath, replacedHtmlJsFile);
        }

        if (newReadFile.includes("getTemplate(")) {
            replaceLineInFile(fullPath, 2, 
                `
                export function getTemplate(content: string) {
                    const policy = trustedTypes.createPolicy('htmlTemplate', {
                        createHTML: (string) => string
                    });
                    return policy.createHTML(\`<!--_html_template_start_-->\${content}<!--_html_template_end_-->\`);
                }`);
        }

        if (newReadFile.includes("function(): HTMLTemplateElement")) {
            const replacedReadFile = newReadFile.replace("function(): HTMLTemplateElement", "function()");
            writeFileSync(fullPath, replacedReadFile);
        }
    }
}