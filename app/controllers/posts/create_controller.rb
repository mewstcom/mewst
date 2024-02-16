# typed: true
# frozen_string_literal: true

class Posts::CreateController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    form = PostForm.new(content: params[:content])

    if form.invalid?
      return render(json: {errors: form.errors.full_messages}, status: :unprocessable_entity)
    end

    result = CreatePostUseCase.new.call(viewer: current_actor!, content: form.content)
    @post = result.post

    render(status: :created)
  end
end
