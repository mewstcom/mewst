# typed: strict
# frozen_string_literal: true

class Dropdowns::ProfileOptionsDropdownComponent < ApplicationComponent
  sig { params(profile: Profile).void }
  def initialize(profile:)
    @profile = profile
  end

  sig { returns(Profile) }
  attr_reader :profile
  private :profile
end
