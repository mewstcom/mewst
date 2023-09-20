# typed: true
# frozen_string_literal: true

class Latest::Me::UpdateController < Latest::ApplicationController
  include Latest::FormErrorable

  def call
    profile = current_profile.not_nil!
    form = Latest::ProfileForm.new(
      profile:,
      atname: params[:atname],
      avatar_url: params[:avatar_url],
      description: params[:description],
      name: params[:name]
    )

    if form.invalid?
      return response_form_errors(resource_class: Latest::FormErrorResource, errors: form.errors)
    end

    result = UpdateProfileUseCase.new.call(
      profile: form.profile.not_nil!,
      atname: form.atname.not_nil!,
      avatar_url: form.avatar_url.not_nil!,
      description: form.description.not_nil!,
      name: form.name.not_nil!
    )

    profile_resource = Latest::ProfileResource.new(profile: result.profile, viewer: profile)
    render(
      json: {
        profile: Latest::ProfileSerializer.new(profile_resource).to_h
      },
      status: :ok
    )
  end
end
