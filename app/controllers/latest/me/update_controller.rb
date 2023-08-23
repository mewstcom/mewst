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

    input = UpdateProfileService::Input.from_latest_form(form:)
    result = ActiveRecord::Base.transaction do
      UpdateProfileService.new.call(input:)
    end

    profile_resource = Latest::ProfileResource.new(profile: result.profile, viewer: profile)
    render(
      json: {
        profile: Latest::ProfileSerializer.new(profile_resource).to_h
      },
      status: :ok
    )
  end
end
