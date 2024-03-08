# typed: strict
# frozen_string_literal: true

class Layouts::BasicLayoutComponent < ApplicationComponent
  sig { params(with_footer: T::Boolean).void }
  def initialize(with_footer: true)
    @with_footer = with_footer
  end

  sig { returns(T::Boolean) }
  attr_reader :with_footer
  private :with_footer
end
