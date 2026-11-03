// Local screenshot QA helper using macOS's bundled AppKit; no image dependency.
// Usage: swift compare.swift reference capture output.png width height
import AppKit

guard CommandLine.arguments.count == 6,
      let reference = NSImage(contentsOfFile: CommandLine.arguments[1]),
      let capture = NSImage(contentsOfFile: CommandLine.arguments[2]),
      let width = Int(CommandLine.arguments[4]),
      let height = Int(CommandLine.arguments[5]) else {
    fatalError("Expected reference, capture, output, width, and height")
}
let gutter = 24, heading = 40
let bitmap = NSBitmapImageRep(bitmapDataPlanes: nil, pixelsWide: width * 2 + gutter,
    pixelsHigh: height + heading, bitsPerSample: 8, samplesPerPixel: 4,
    hasAlpha: true, isPlanar: false, colorSpaceName: .deviceRGB, bytesPerRow: 0, bitsPerPixel: 0)!
NSGraphicsContext.saveGraphicsState()
NSGraphicsContext.current = NSGraphicsContext(bitmapImageRep: bitmap)
NSGraphicsContext.current?.imageInterpolation = .high
NSColor.white.setFill()
NSBezierPath(rect: NSRect(x: 0, y: 0, width: width * 2 + gutter, height: height + heading)).fill()
for (index, image) in [reference, capture].enumerated() {
    let scale = min(CGFloat(width) / image.size.width, CGFloat(height) / image.size.height)
    let size = NSSize(width: image.size.width * scale, height: image.size.height * scale)
    let x = CGFloat(index * (width + gutter)) + (CGFloat(width) - size.width) / 2
    image.draw(in: NSRect(x: x, y: CGFloat(height) - size.height, width: size.width, height: size.height))
    let label = index == 0 ? "Selected reference (normalized)" : "Browser-rendered implementation"
    (label as NSString).draw(at: NSPoint(x: CGFloat(index * (width + gutter)) + 8, y: CGFloat(height) + 12),
        withAttributes: [.font: NSFont.systemFont(ofSize: 13), .foregroundColor: NSColor.black])
}
NSGraphicsContext.restoreGraphicsState()
try bitmap.representation(using: .png, properties: [:])!.write(to: URL(fileURLWithPath: CommandLine.arguments[3]))
