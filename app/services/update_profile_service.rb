# typed: strict
# frozen_string_literal: true

class UpdateProfileService < ApplicationService
  class Input < T::Struct
    extend T::Sig

    const :profile, Profile
    const :atname, String
    const :avatar_url, String
    const :description, String
    const :name, String

    sig { params(form: Latest::ProfileForm).returns(Input) }
    def self.from_latest_form(form:)
      new(
        profile: form.profile.not_nil!,
        atname: form.atname.not_nil!,
        avatar_url: form.avatar_url.not_nil!,
        description: form.description.not_nil!,
        name: form.name.not_nil!
      )
    end
  end

  class Result < T::Struct
    const :profile, Profile
  end

  sig { params(input: Input).returns(Result) }
  def call(input:)
    input.profile.update!(
      atname: input.atname,
      avatar_url: input.avatar_url,
      description: input.description,
      name: input.name
    )

    Result.new(profile: input.profile)
  end
end
