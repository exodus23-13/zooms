require 'zooms/rails'

class CustomPlan < Zooms::Rails

  # def my_custom_command
  #  # see https://github.com/exodus23-13/zooms/blob/master/docs/ruby/modifying.md
  # end

end

Zooms.plan = CustomPlan.new
