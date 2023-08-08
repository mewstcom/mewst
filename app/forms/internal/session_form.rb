# typed: strict
# frozen_string_literal: true

class Internal::SessionForm < Internal::ApplicationForm
  attribute :email, :string
  attribute :password, :string

  validates :email, email: true, presence: true
  validates :password, presence: true
  validates :user, presence: true
  validate :authentication

  sig { returns(User) }
  def user!
    T.cast(user, User)
  end

  private

  sig { returns(T.nilable(User)) }
  def user
    @user ||= T.let(User.find_by(email:), T.nilable(User))
  end

  sig { void }
  def authentication
    unless user!.authenticate(password)
      errors.add(:base, :unauthenticated)
    end
  end
end
