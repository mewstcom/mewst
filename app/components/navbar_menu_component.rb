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
    "h-[40px] hover:bg-gray-100 rounded-full text-center w-[40px]"
  end

  sig { params(icon_name: String).returns(String) }
  private def icon_class_name(icon_name)
    class_names("bi leading-[40px] text-2xl", "bi-#{icon_name}")
  end
end
