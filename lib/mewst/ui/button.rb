# typed: strict
# frozen_string_literal: true

class Mewst::UI::Button < Mewst::UI::Base
  sig { params(as: Symbol, class_name: String, options: T::Hash[Symbol, String]).void }
  def initialize(as: :button, class_name: "", **options)
    @as = as
    @class_name = class_name
    @options = options
  end

  sig { returns(Symbol) }
  attr_reader :as
  private :as

  sig { returns(String) }
  attr_reader :class_name
  private :class_name

  sig { returns(T::Hash[Symbol, String]) }
  attr_reader :options
  private :options
end
