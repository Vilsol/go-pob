export const wrapText = (text: string, context: CanvasRenderingContext2D, width: number): string[] => {
  const result = [];

  let currentWord = '';
  text.split(' ').forEach((word) => {
    if (context.measureText(currentWord + word).width < width) {
      currentWord += ' ' + word;
    } else {
      result.push(currentWord.trim());
      currentWord = word;
    }
  });

  if (currentWord.length > 0) {
    result.push(currentWord.trim());
  }

  return result;
};
