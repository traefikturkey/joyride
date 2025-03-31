Dir.glob(File.join(__dir__, '*.handler')).each { |file| require_relative file }
