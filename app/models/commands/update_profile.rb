# typed: strict
# frozen_string_literal: true

class Commands::UpdateProfile < Commands::Base
  class Result < T::Struct
    const :profile, Profile
  end

  sig { params(form: Forms::Profile).void }
  def initialize(form:)
    @form = form
  end

  sig { returns(Result) }
  def call
    form.profile!.update!(form.attributes)

    Result.new(profile: form.profile!)
  end

  private

  sig { returns(Forms::Profile) }
  attr_reader :form
end
