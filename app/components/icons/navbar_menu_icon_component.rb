# typed: strict
# frozen_string_literal: true

class Icons::NavbarMenuIconComponent < ApplicationComponent
  sig { params(name: String, active_name: String, size: String, active: T::Boolean).void }
  def initialize(name:, active_name:, size:, active:)
    @name = name
    @active_name = active_name
    @size = size
    @active = active
  end

  sig { returns(String) }
  attr_reader :name
  private :name

  sig { returns(String) }
  attr_reader :active_name
  private :active_name

  sig { returns(String) }
  attr_reader :size
  private :size

  sig { returns(T::Boolean) }
  attr_reader :active
  private :active
end
