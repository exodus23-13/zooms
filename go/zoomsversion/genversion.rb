require 'fileutils'

version = File.read('../../VERSION').chomp

File.open('zoomsversion.go', 'w') { |f| f.puts <<END
package zoomsversion

const VERSION string = "#{version}"
END
}
