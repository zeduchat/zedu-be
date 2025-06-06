package recentLogs

import (
    "bufio"
    "bytes"
    "io"
    "os"
)


func TailLines(filePath string, n int) (string, error) {
    file, err := os.Open(filePath)
    if err != nil {
        return "", err
    }
    defer file.Close()

    const readBlockSize = 1024
    stat, err := file.Stat()
    if err != nil {
        return "", err
    }

    size := stat.Size()
    var buffer bytes.Buffer
    var lineCount int

    offset := int64(0)
    var chunk []byte

    for {
        if size-int64(readBlockSize)-offset < 0 {
            offset = size
        } else {
            offset += int64(readBlockSize)
        }

        start := size - offset
        if start < 0 {
            start = 0
        }

        length := offset
        if start+length > size {
            length = size - start
        }

        chunk = make([]byte, length)
        _, err := file.ReadAt(chunk, start)
        if err != nil && err != io.EOF {
            return "", err
        }

        buffer.Write(chunk)
        lineCount = bytes.Count(buffer.Bytes(), []byte("\n"))

        if lineCount > n || start == 0 {
            break
        }
    }

    lines := make([]string, 0, n)
    scanner := bufio.NewScanner(bytes.NewReader(buffer.Bytes()))
    for scanner.Scan() {
        lines = append(lines, scanner.Text())
    }

    if len(lines) > n {
        lines = lines[len(lines)-n:]
    }

    return string(bytes.Join(toByteSlices(lines), []byte("\n"))), nil
}


func toByteSlices(lines []string) [][]byte {
    result := make([][]byte, len(lines))
    for i, line := range lines {
        result[i] = []byte(line)
    }
    return result
}
