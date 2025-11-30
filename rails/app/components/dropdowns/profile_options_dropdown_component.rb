# typed: strict
# frozen_string_literal: true

class Dropdowns::ProfileOptionsDropdownComponent < ApplicationComponent
  sig { params(profile: ProfileRecord).void }
  def initialize(profile:)
    @profile = profile
  end

  sig { returns(ProfileRecord) }
  attr_reader :profile
  private :profile
end
