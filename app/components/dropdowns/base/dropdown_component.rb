# typed: strict
# frozen_string_literal: true

class Dropdowns::Base::DropdownComponent < ApplicationComponent
  sig { params(class_name: String).void }
  def initialize(class_name: "")
    @class_name = class_name
  end

  sig { returns(String) }
  private def class_name
    class_list = %w[dropdown]
    class_list << @class_name if @class_name.present?
    class_list.join(" ")
  end
end
