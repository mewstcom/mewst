# typed: true
# frozen_string_literal: true

class Latest::Reposts::CreateController < Latest::ApplicationController
  def call
    form = Forms::Repost.new(
      profile: current_profile!,
      target_post_id: params[:post_id]
    )

    if form.invalid?
      return render(
        json: Latest::Presenters::RepostFormErrors.new(errors: form.errors),
        status: :unprocessable_entity
      )
    end

    result = ActiveRecord::Base.transaction do
      Services::CreateRepost.new(form:).call
    end

    render(
      json: Latest::Presenters::Repost.new(post: result.post),
      status: :created
    )
  end
end
