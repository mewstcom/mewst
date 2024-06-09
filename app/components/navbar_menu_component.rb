# typed: strict
# frozen_string_literal: true

class NavbarMenuComponent < ApplicationComponent
  sig { params(class_name: String).void }
  def initialize(class_name: "")
    @class_name = class_name
  end

  sig { returns(String) }
  attr_reader :class_name
  private :class_name

  sig { returns(String) }
  private def link_class_name
    "hover:bg-gray-100 p-1 rounded-full"
  end

  sig { returns(String) }
  def link_with_text_class_name
    "hover:bg-gray-100 rounded-lg text-center w-[100px]"
  end

  sig { returns(String) }
  private def text_class_name
    "mt-1 text-gray-500 text-xs"
  end
end
