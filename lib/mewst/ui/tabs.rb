# typed: strict
# frozen_string_literal: true

class Mewst::UI::Tabs < Mewst::UI::Base
  renders_many :items, Mewst::UI::Tabs::Item

  sig { params(class_name: String).void }
  def initialize(class_name: "")
    @class_name = class_name
  end

  sig { returns(String) }
  attr_reader :class_name
  private :class_name
end
